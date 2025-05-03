package controllers

import (
	"context"
	cryptorand "crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"makerthon/models"
	"makerthon/views"
	"makerthon/views/components"
	"math/rand"
	"net/http"
	"net/smtp"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
	"golang.org/x/oauth2/google"
	"gorm.io/gorm"
)

type authController struct {
	db           *gorm.DB
	oauthConfigs map[string]*oauth2.Config
}

func NewAuthController(db *gorm.DB) *authController {
	assertDatabase(db, "auth")

	baseURL := os.Getenv("BASE_URL")
	googleClientID := os.Getenv("GOOGLE_CLIENT_ID")
	googleClientSecret := os.Getenv("GOOGLE_CLIENT_SECRET")
	githubClientID := os.Getenv("GITHUB_CLIENT_ID")
	githubClientSecret := os.Getenv("GITHUB_CLIENT_SECRET")

	log.Printf("Base URL: %s", baseURL)
	log.Printf("Google Client ID: %s", googleClientID)
	log.Printf("GitHub Client ID: %s", githubClientID)

	oauthConfigs := map[string]*oauth2.Config{
		"google": {
			ClientID:     googleClientID,
			ClientSecret: googleClientSecret,
			RedirectURL:  baseURL + "/auth/oauth/google/callback",
			Scopes:       []string{"https://www.googleapis.com/auth/userinfo.email", "https://www.googleapis.com/auth/userinfo.profile"},
			Endpoint:     google.Endpoint,
		},
		"github": {
			ClientID:     githubClientID,
			ClientSecret: githubClientSecret,
			RedirectURL:  baseURL + "/auth/oauth/github/callback",
			Scopes:       []string{"user:email"},
			Endpoint:     github.Endpoint,
		},
	}

	return &authController{
		db:           db,
		oauthConfigs: oauthConfigs,
	}
}

func (ac *authController) LoadRoutes(router *gin.RouterGroup) {
	assertRouteGroup(router, "auth")

	router.GET("/login", ac.LoginPage)
	router.GET("/register", ac.RegisterPage)
	router.GET("/verify", ac.ShowVerificationForm)

	router.POST("/login", ac.Login)
	router.POST("/register", ac.Register)
	router.POST("/verify", ac.VerifyEmail)
	router.POST("/resend-verification", ac.ResendVerification)

	router.GET("/oauth/:provider", ac.OAuthLogin)
	router.GET("/oauth/:provider/callback", ac.OAuthCallback)
	router.GET("/logout", ac.Logout)
}

func (ac *authController) LoginPage(ctx *gin.Context) {
	views.Login().Render(ctx, ctx.Writer)
}

func (ac *authController) RegisterPage(ctx *gin.Context) {
	views.Register().Render(ctx, ctx.Writer)
}

func (ac *authController) Login(ctx *gin.Context) {
	username := ctx.PostForm("username")
	password := ctx.PostForm("password")

	var user models.User
	if err := ac.db.Where("username = ?", username).First(&user).Error; err != nil {
		ctx.Set("error", "Tên đăng nhập hoặc mật khẩu không chính xác")
		views.Login().Render(ctx, ctx.Writer)
		return
	}

	if user.Password == "" || bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)) != nil {
		ctx.Set("error", "Tên đăng nhập hoặc mật khẩu không chính xác")
		views.Login().Render(ctx, ctx.Writer)
		return
	}

	ctx.SetCookie("user_id", fmt.Sprintf("%d", user.ID), 3600*24*30, "/", "", false, true)
	ctx.Redirect(http.StatusFound, "/")
}

func (ac *authController) Register(ctx *gin.Context) {
	username := ctx.PostForm("username")
	email := ctx.PostForm("email")
	password := ctx.PostForm("password")
	var count int64
	ac.db.Model(&models.User{}).Where("username = ? OR email = ?", username, email).Count(&count)
	if count > 0 {
		ctx.Set("error", "Tên đăng nhập hoặc email đã được sử dụng")
		views.Register().Render(ctx, ctx.Writer)
		return
	}
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		ctx.Set("error", "Lỗi khi đăng ký, vui lòng thử lại")
		views.Register().Render(ctx, ctx.Writer)
		return
	}
	verificationCode := ac.generateVerificationCode()
	expiryTime := time.Now().Add(10 * time.Minute)

	user := models.User{
		Username:         username,
		Email:            email,
		Password:         string(hashedPassword),
		IsActive:         false,
		Token:            100,
		VerificationCode: verificationCode,
		CodeExpiry:       &expiryTime,
		EmailVerified:    false,
	}

	if err := ac.db.Create(&user).Error; err != nil {
		ctx.Set("error", "Lỗi khi đăng ký, vui lòng thử lại")
		views.Register().Render(ctx, ctx.Writer)
		return
	}

	go ac.sendVerificationEmail(user.Username, user.Email, verificationCode)

	log.Printf("User registered successfully. Redirecting to verification page with email: %s", email)
	ctx.Redirect(http.StatusFound, "/auth/verify?email="+email+"&new=true")
}

func (ac *authController) generateVerificationCode() string {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	return fmt.Sprintf("%06d", r.Intn(1000000))
}

func (ac *authController) sendVerificationEmail(username, email, code string) {
	verificationURL := os.Getenv("VERIFICATION_URL") + "?email=" + email

	var builder strings.Builder
	components.VerificationEmailTemplate(username, code, 10, verificationURL).Render(context.Background(), &builder)
	htmlBody := builder.String()

	err := ac.sendEmail(email, "Xác thực tài khoản QuizCreator của bạn", htmlBody)
	if err != nil {
		log.Printf("Error sending verification email to %s: %v", email, err)
	} else {
		log.Printf("Verification email sent to %s with code %s", email, code)
	}
}
func (ac *authController) sendEmail(to, subject, htmlBody string) error {
	from := os.Getenv("SMTP_FROM")
	password := os.Getenv("SMTP_PASSWORD")
	smtpHost := os.Getenv("SMTP_HOST")
	smtpPort := os.Getenv("SMTP_PORT")
	headers := make(map[string]string)
	headers["From"] = from
	headers["To"] = to
	headers["Subject"] = subject
	headers["MIME-Version"] = "1.0"
	headers["Content-Type"] = "text/html; charset=UTF-8"

	var message string
	for k, v := range headers {
		message += fmt.Sprintf("%s: %s\r\n", k, v)
	}
	message += "\r\n" + htmlBody
	auth := smtp.PlainAuth("", from, password, smtpHost)
	err := smtp.SendMail(smtpHost+":"+smtpPort, auth, from, []string{to}, []byte(message))
	if err != nil {
		return err
	}

	return nil
}

func (ac *authController) Logout(ctx *gin.Context) {
	ctx.SetCookie("user_id", "", -1, "/", "", false, true)
	ctx.Redirect(http.StatusFound, "/")
}

func (ac *authController) OAuthLogin(ctx *gin.Context) {
	provider := ctx.Param("provider")
	config, exists := ac.oauthConfigs[provider]
	if !exists {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Không hỗ trợ OAuth provider này"})
		return
	}
	state := ac.generateRandomString(32)
	ctx.SetCookie("oauth_state", state, 60*15, "/", "", false, true)

	authURL := config.AuthCodeURL(state)
	ctx.Redirect(http.StatusFound, authURL)
}

func (ac *authController) OAuthCallback(ctx *gin.Context) {
	provider := ctx.Param("provider")
	config, exists := ac.oauthConfigs[provider]
	if !exists {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Không hỗ trợ OAuth provider này"})
		return
	}
	expectedState, _ := ctx.Cookie("oauth_state")
	receivedState := ctx.Query("state")

	if expectedState == "" || expectedState != receivedState {
		log.Printf("Invalid OAuth state. Expected: %s, Received: %s", expectedState, receivedState)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid OAuth state"})
		return
	}

	ctx.SetCookie("oauth_state", "", -1, "/", "", false, true)

	code := ctx.Query("code")
	token, err := config.Exchange(ctx.Request.Context(), code)
	if err != nil {
		log.Printf("Error exchanging code for token: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể xác thực với OAuth provider"})
		return
	}

	userInfo, err := ac.getUserInfoFromProvider(ctx.Request.Context(), provider, token.AccessToken)
	if err != nil {
		log.Printf("Error getting user info: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lấy thông tin người dùng từ OAuth provider"})
		return
	}

	var user models.User
	result := ac.db.Where("oauth_provider = ? AND oauth_id = ?", provider, userInfo.ID).First(&user)

	if result.Error != nil {
		var emailUser models.User
		emailResult := ac.db.Where("email = ?", userInfo.Email).First(&emailUser)

		if emailResult.Error == nil {
			log.Printf("Found existing user with email %s, linking OAuth account", userInfo.Email)
			emailUser.OAuthProvider = provider
			emailUser.OAuthID = userInfo.ID
			emailUser.OAuthToken = token.AccessToken
			ac.db.Save(&emailUser)

			ctx.SetCookie("user_id", fmt.Sprintf("%d", emailUser.ID), 3600*24*30, "/", "", false, true)
			ctx.Redirect(http.StatusFound, "/")
			return
		} else {
			username := userInfo.Username
			if username == "" {
				username = userInfo.Email
			}
			user = models.User{
				Username:      username,
				Email:         userInfo.Email,
				Password:      "",
				IsActive:      true,
				Token:         100,
				OAuthProvider: provider,
				OAuthID:       userInfo.ID,
				OAuthToken:    token.AccessToken,
			}

			if err := ac.db.Create(&user).Error; err != nil {
				log.Printf("Error creating user: %v", err)
				ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tạo tài khoản mới"})
				return
			}
			log.Printf("Created new user for %s with OAuth provider %s", userInfo.Email, provider)
		}
	} else {
		user.OAuthToken = token.AccessToken
		ac.db.Save(&user)
		log.Printf("Updated OAuth token for existing user %s", user.Email)
	}
	ctx.SetCookie("user_id", fmt.Sprintf("%d", user.ID), 3600*24*30, "/", "", false, true) // 30 ngày
	ctx.Redirect(http.StatusFound, "/")
}

type OAuthUserInfo struct {
	ID       string
	Username string
	Email    string
	Name     string
}

func (ac *authController) getUserInfoFromProvider(ctx context.Context, provider, accessToken string) (*OAuthUserInfo, error) {
	switch provider {
	case "google":
		return ac.getGoogleUserInfo(ctx, accessToken)
	case "github":
		return ac.getGithubUserInfo(ctx, accessToken)
	default:
		return nil, fmt.Errorf("unsupported provider: %s", provider)
	}
}

func (ac *authController) getGoogleUserInfo(ctx context.Context, accessToken string) (*OAuthUserInfo, error) {
	client := &http.Client{}
	req, err := http.NewRequestWithContext(ctx, "GET", "https://www.googleapis.com/oauth2/v3/userinfo", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", accessToken))

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get user info: %s", resp.Status)
	}

	var result struct {
		Sub           string `json:"sub"`
		Email         string `json:"email"`
		EmailVerified bool   `json:"email_verified"`
		Name          string `json:"name"`
		GivenName     string `json:"given_name"`
		FamilyName    string `json:"family_name"`
		Picture       string `json:"picture"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &OAuthUserInfo{
		ID:       result.Sub,
		Username: result.Name,
		Email:    result.Email,
		Name:     result.Name,
	}, nil
}

func (ac *authController) getGithubUserInfo(ctx context.Context, accessToken string) (*OAuthUserInfo, error) {
	client := &http.Client{}
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.github.com/user", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", fmt.Sprintf("token %s", accessToken))
	req.Header.Add("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get user info: %s", resp.Status)
	}

	var result struct {
		ID        int    `json:"id"`
		Login     string `json:"login"`
		Name      string `json:"name"`
		Email     string `json:"email"`
		AvatarURL string `json:"avatar_url"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	email := result.Email
	if email == "" {
		email, _ = ac.getGithubEmail(ctx, accessToken)
	}

	return &OAuthUserInfo{
		ID:       fmt.Sprintf("%d", result.ID),
		Username: result.Login,
		Email:    email,
		Name:     result.Name,
	}, nil
}

func (ac *authController) getGithubEmail(ctx context.Context, accessToken string) (string, error) {
	client := &http.Client{}
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.github.com/user/emails", nil)
	if err != nil {
		return "", err
	}
	req.Header.Add("Authorization", fmt.Sprintf("token %s", accessToken))
	req.Header.Add("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to get emails: %s", resp.Status)
	}

	var emails []struct {
		Email    string `json:"email"`
		Primary  bool   `json:"primary"`
		Verified bool   `json:"verified"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&emails); err != nil {
		return "", err
	}

	for _, email := range emails {
		if email.Primary && email.Verified {
			return email.Email, nil
		}
	}
	for _, email := range emails {
		if email.Verified {
			return email.Email, nil
		}
	}
	if len(emails) > 0 {
		return emails[0].Email, nil
	}

	return "", fmt.Errorf("no email found")
}

func (ac *authController) generateRandomString(length int) string {
	b := make([]byte, length)
	cryptorand.Read(b)
	return fmt.Sprintf("%x", b)
}

func (ac *authController) ShowVerificationForm(ctx *gin.Context) {
	email := ctx.Query("email")
	if email == "" {
		ctx.Redirect(http.StatusFound, "/auth/login")
		return
	}
	var user models.User
	if err := ac.db.Where("email = ?", email).First(&user).Error; err != nil {
		ctx.Redirect(http.StatusFound, "/auth/register")
		return
	}

	views.VerifyEmail(email).Render(ctx, ctx.Writer)
}

func (ac *authController) VerifyEmail(ctx *gin.Context) {
	email := ctx.PostForm("email")
	code := ctx.PostForm("code")
	var user models.User
	if err := ac.db.Where("email = ?", email).First(&user).Error; err != nil {
		ctx.Set("error", "Email không tồn tại trong hệ thống")
		views.VerifyEmail(email).Render(ctx, ctx.Writer)
		return
	}
	if user.VerificationCode != code {
		ctx.Set("error", "Mã xác thực không chính xác")
		views.VerifyEmail(email).Render(ctx, ctx.Writer)
		return
	}
	if user.CodeExpiry == nil || time.Now().After(*user.CodeExpiry) {
		ctx.Set("error", "Mã xác thực đã hết hạn")
		views.VerifyEmail(email).Render(ctx, ctx.Writer)
		return
	}
	user.EmailVerified = true
	user.IsActive = true
	user.VerificationCode = ""
	user.CodeExpiry = nil
	if err := ac.db.Save(&user).Error; err != nil {
		ctx.Set("error", "Lỗi khi kích hoạt tài khoản")
		views.VerifyEmail(email).Render(ctx, ctx.Writer)
		return
	}
	ctx.SetCookie("user_id", fmt.Sprintf("%d", user.ID), 3600*24*30, "/", "", false, true)
	ctx.Redirect(http.StatusFound, "/")
}

func (ac *authController) ResendVerification(ctx *gin.Context) {
	email := ctx.PostForm("email")

	var user models.User
	if err := ac.db.Where("email = ?", email).First(&user).Error; err != nil {
		ctx.Set("error", "Email không tồn tại")
		views.VerifyEmail(email).Render(ctx, ctx.Writer)
		return
	}

	if user.EmailVerified {
		ctx.Redirect(http.StatusFound, "/auth/login")
		return
	}
	verificationCode := ac.generateVerificationCode()
	expiryTime := time.Now().Add(10 * time.Minute)

	user.VerificationCode = verificationCode
	user.CodeExpiry = &expiryTime
	ac.db.Save(&user)
	go ac.sendVerificationEmail(user.Username, user.Email, verificationCode)

	ctx.Set("success", "Mã xác thực mới đã được gửi đến email của bạn")
	views.VerifyEmail(email).Render(ctx, ctx.Writer)
}
