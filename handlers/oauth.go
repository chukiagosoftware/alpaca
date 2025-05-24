package handlers

import (
    "encoding/json"
    "net/http"
	"os"
    "time"

    "github.com/edamsoft-sre/alpaca/models"
    "github.com/edamsoft-sre/alpaca/repository"
    "github.com/golang-jwt/jwt"
    "github.com/markbates/goth/gothic"
    "github.com/markbates/goth"
	"github.com/markbates/goth/providers/google"
	
    
)

func InitGoth() {
    goth.UseProviders(
        google.New(
            os.Getenv("GOOGLE_CLIENT_ID"),
            os.Getenv("GOOGLE_CLIENT_SECRET"),
            os.Getenv("GOOGLE_REDIRECT_URL"),
            "email", "profile",
        ),
    )
}

func GothLoginHandler(w http.ResponseWriter, r *http.Request) {
    gothic.BeginAuthHandler(w, r)
}

func GothCallbackHandler(w http.ResponseWriter, r *http.Request) {
    user, err := gothic.CompleteUserAuth(w, r)
    if err != nil {
        http.Error(w, "OAuth failed: "+err.Error(), http.StatusInternalServerError)
        return
    }

    // Find or create user in your DB
    dbUser, err := repository.GetUserByEmail(r.Context(), user.Email)
    if dbUser == nil {
        dbUser = &models.User{
            Email: user.Email,
            Id:    user.UserID, // or generate your own
        }
        _ = repository.InsertUser(r.Context(), dbUser)
    }

    // Issue JWT
    claims := models.AppClaims{
        UserId: dbUser.Id,
        StandardClaims: jwt.StandardClaims{
            ExpiresAt: time.Now().Add(2 * 24 * time.Hour).Unix(),
        },
    }
    jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    tokenString, err := jwtToken.SignedString([]byte(os.Getenv("JWT_SECRET")))
    if err != nil {
        http.Error(w, "Failed to sign JWT: "+err.Error(), http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(LoginResponse{Token: tokenString})
}