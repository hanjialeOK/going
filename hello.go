package main

import (
	"fmt"
	"net/http"
	"net/mail"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type UserInfo struct {
	ID       int    `form:"id"`
	Name     string `form:"name" binding:"required"`
	Email    string `form:"email" binding:"required"`
	Password string `form:"password" binding:"required"`
}

var jwtKey = []byte("my_secret_key")

func createToken(email string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"email": email,
		"exp":   time.Now().Add(time.Minute * 10).Unix(),
	})

	return token.SignedString(jwtKey)
}

func validateToken(tokenString string) (*jwt.Token, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}
		return jwtKey, nil
	})

	if err != nil {
		return nil, err
	}

	if _, ok := token.Claims.(jwt.MapClaims); !ok || !token.Valid {
		return nil, fmt.Errorf("bad token.")
	}

	return token, nil
}

func validateCookie(c *gin.Context) (*jwt.Token, error) {
	tokenString, err := c.Cookie("gin_cookie")
	if err != nil {
		return nil, err
	}
	token, err := validateToken(tokenString)
	if err != nil {
		return nil, err
	}
	return token, nil
}

func main() {
	router := gin.Default()

	router.POST("/account/create", create)
	router.POST("/account/login", login)
	router.POST("/account/logout", logout)
	router.POST("/account/update", update)
	router.POST("/account/delete", delete)
	router.POST("/account/showall", showall)

	router.Run(":8001")
}

func valid(email string) bool {
	_, err := mail.ParseAddress(email)
	return err == nil
}

func create(c *gin.Context) {
	// handle login
	var form UserInfo
	if err := c.ShouldBind(&form); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// TODO: validate the email
	if valid(form.Email) == false {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad email."})
		return
	}

	dsn := "root:123456@tcp(127.0.0.1:3306)/gobase"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		// c.String(http.StatusBadRequest, "open mysql fail")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// check whether the name exists.
	var user UserInfo
	db.Table("account").Where("name = ?", form.Name).First(&user)
	if (user != UserInfo{}) {
		// c.String(http.StatusBadRequest, fmt.Sprintf("create failed. %s exists!", form.Name))
		c.JSON(http.StatusBadRequest, gin.H{"error": "name exists!"})
		return
	}
	// check whether the email has been used.
	db.Table("account").Where("email = ?", form.Email).First(&user)
	if (user != UserInfo{}) {
		// c.String(http.StatusBadRequest, fmt.Sprintf("create failed. email %s has been used!", form.Email))
		c.JSON(http.StatusBadRequest, gin.H{"error": "email has been used!"})
		return
	}
	// insert the new account.
	user = UserInfo{Name: form.Name, Email: form.Email, Password: form.Password}
	result := db.Table("account").Create(&user)
	if result.Error != nil {
		// c.String(http.StatusBadRequest, "create accout fail.")
		c.JSON(http.StatusBadRequest, gin.H{"error": result.Error})
		return
	}
	// tell the user it's ok.
	c.String(http.StatusOK, fmt.Sprintf("OK! Add '%s'!", form.Name))
}

func login(c *gin.Context) {
	email := c.Query("email")
	password := c.Query("password")

	// TODO: validate the email
	if valid(email) == false {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad email."})
		return
	}

	dsn := "root:123456@tcp(127.0.0.1:3306)/gobase"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// retrieve the user by email.
	var user UserInfo
	db.Table("account").Where("email = ?", email).First(&user)
	if (user == UserInfo{}) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "this email has not been registered."})
		return
	}
	// validate the password
	if user.Password != password {
		c.JSON(http.StatusBadRequest, gin.H{"error": "wrong password."})
		return
	}
	// Create the JWT string.
	token, err := createToken(user.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create token"})
		return
	}
	// tell the user it's ok.
	c.SetCookie("gin_cookie", token, 3600, "/", "", false, true)
	c.String(http.StatusOK, fmt.Sprintf("welcome %s!", user.Name))
}

func logout(c *gin.Context) {
	c.SetCookie("gin_cookie", "logout", 0, "/", "", false, true)
	c.String(http.StatusOK, fmt.Sprintf("bye."))
}

func update(c *gin.Context) {
	token, err := validateCookie(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// get email from cookie
	email := token.Claims.(jwt.MapClaims)["email"]
	// new password
	password := c.Query("password")
	// update
	dsn := "root:123456@tcp(127.0.0.1:3306)/gobase"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// retrieve the user by email.
	var user UserInfo
	db.Table("account").Where("email = ?", email).First(&user)
	if (user == UserInfo{}) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "this email has not been registered."})
		return
	}
	user.Password = password
	db.Table("account").Save(&user)
	c.String(http.StatusOK, fmt.Sprintf("password updated!"))
}

func delete(c *gin.Context) {
	email := c.Query("email")
	password := c.Query("password")

	// TODO: validate the email
	if valid(email) == false {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad email."})
		return
	}

	dsn := "root:123456@tcp(127.0.0.1:3306)/gobase"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// retrieve the user by email.
	var user UserInfo
	db.Table("account").Where("email = ?", email).First(&user)
	if (user == UserInfo{}) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "this email has not been registered."})
		return
	}
	// validate the password
	if user.Password != password {
		c.JSON(http.StatusBadRequest, gin.H{"error": "wrong password."})
		return
	}
	db.Table("account").Where("id = ?", user.ID).Delete(&user)
	// tell the user it's ok.
	c.String(http.StatusOK, fmt.Sprintf("delete '%s'.", user.Name))
}

func showall(c *gin.Context) {
	if _, err := validateCookie(c); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// show all
	dsn := "root:123456@tcp(127.0.0.1:3306)/gobase"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// retrieve all users.
	var users []UserInfo
	db.Table("account").Find(&users)
	var names []string
	for i := 0; i < len(users); i++ {
		names = append(names, users[i].Name)
	}
	// list all users.
	c.String(http.StatusOK, fmt.Sprintf("'%s'", names))
}
