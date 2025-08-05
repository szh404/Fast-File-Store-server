package controller

import (
	"encoding/json"
	"file-store/lib"
	"file-store/model"
	"file-store/util"
	"fmt"
	"github.com/gin-gonic/gin"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"
)

type GithubAccessTokenResp struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	Scope       string `json:"scope"`
}

type GithubUserInfo struct {
	ID        int64  `json:"id"`
	Login     string `json:"login"`
	Name      string `json:"name"`
	AvatarUrl string `json:"avatar_url"`
}

// 登录页
func Login(c *gin.Context) {
	c.HTML(http.StatusOK, "login.html", nil)
}

// 跳转到 GitHub 授权页面
func HandlerGithubLogin(c *gin.Context) {
	conf := lib.LoadServerConfig()
	state := "xxxxxx" // 可随机生成

	redirectURI := url.QueryEscape(conf.RedirectURI)
	authURL := fmt.Sprintf(
		"https://github.com/login/oauth/authorize?client_id=%s&redirect_uri=%s&scope=user&state=%s",
		conf.Client_Id, redirectURI, state,
	)

	c.Redirect(http.StatusFound, authURL)
}

// GitHub 回调，获取 access_token 并获取用户信息
func GetGithubToken(c *gin.Context) {
	conf := lib.LoadServerConfig()
	code := c.Query("code")

	tokenURL := "https://github.com/login/oauth/access_token"
	data := url.Values{}
	data.Set("client_id", conf.Client_Id)
	data.Set("client_secret", conf.Client_Key)
	data.Set("code", code)
	fmt.Println("Client_ID =", conf.Client_Id)
	fmt.Println("Client_Secret =", conf.Client_Key)

	req, err := http.NewRequest("POST", tokenURL, nil)
	if err != nil {
		c.String(http.StatusInternalServerError, "创建请求失败: %v", err)
		return
	}
	req.URL.RawQuery = data.Encode()
	req.Header.Set("Accept", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		c.String(http.StatusInternalServerError, "请求失败: %v", err)
		return
	}
	defer resp.Body.Close()

	var tokenResp GithubAccessTokenResp
	body, _ := ioutil.ReadAll(resp.Body)
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		c.String(http.StatusInternalServerError, "解析 access_token 失败: %v", err)
		return
	}

	if tokenResp.AccessToken == "" {
		c.String(http.StatusInternalServerError, "获取 access_token 失败: %s", string(body))
		return
	}

	getGithubUserInfo(tokenResp.AccessToken, c)
}

func getGithubUserInfo(accessToken string, c *gin.Context) {
	userInfoURL := "https://api.github.com/user"

	req, _ := http.NewRequest("GET", userInfoURL, nil)
	req.Header.Set("Authorization", "token "+ accessToken)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		c.String(http.StatusInternalServerError, "获取 GitHub 用户信息失败: %v", err)
		return
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	var user GithubUserInfo
	if err := json.Unmarshal(body, &user); err != nil {
		c.String(http.StatusInternalServerError, "解析 GitHub 用户信息失败: %v", err)
		return
	}

	loginSucceedGithub(user, c)
}

// 登录成功，处理逻辑
func loginSucceedGithub(user GithubUserInfo, c *gin.Context) {
	openId := fmt.Sprintf("github_%d", user.ID)
	fmt.Println("GitHub OpenID:", openId)

	// 创建 token 并保存
	token := util.EncodeMd5("token" + string(time.Now().Unix()) + openId)
	fmt.Println("生成的 token:", token)
	if err := lib.SetKey(token, openId, 24*3600); err != nil {
		c.String(http.StatusInternalServerError, "Redis 保存 token 失败: %v", err)
		return
	}

	c.SetCookie("Token", token, 3600*24, "/", "pyxgo.cn", false, true)

	if ok := model.QueryUserExists(openId); ok {
		c.Redirect(http.StatusMovedPermanently, "/cloud/index")
	} else {
		model.CreateUser(openId, user.Login, user.AvatarUrl)
		c.Redirect(http.StatusMovedPermanently, "/cloud/index")
	}
}

// 退出登录
func Logout(c *gin.Context) {
	token, err := c.Cookie("Token")
	if err != nil {
		fmt.Println("cookie", err.Error())
	}
	if err := lib.DelKey(token); err != nil {
		fmt.Println("Del Redis Err:", err.Error())
	}

	c.SetCookie("Token", "", 0, "/", "pyxgo.cn", false, false)
	c.Redirect(http.StatusFound, "/")
}
