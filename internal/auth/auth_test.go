package auth

import (
	"time"
	"github.com/google/uuid"
	"testing"
)

func TestGood(t *testing.T) {
	userID := uuid.New()
	tokenSecret := "kronos"
	expiresIn := 10 * time.Minute
	tokenString, err := MakeJWT(userID, tokenSecret, expiresIn)
	if err != nil {
		t.Log(err.Error()+ "\n")
		t.Fail()
	}
	tokenID, err := ValidateJWT(tokenString, tokenSecret)
	if err != nil {
		t.Log(err.Error()+ "\n")
		t.Fail()
	}
	if userID != tokenID {
		t.Log("wrong ID")
		t.Fail()
	}
}

func TestExpired(t *testing.T) {
	userID := uuid.New()
	tokenSecret := "rhea"
	expiresIn := 10 * time.Millisecond
	tokenString, err := MakeJWT(userID, tokenSecret, expiresIn)
	if err != nil {
		t.Log(err.Error()+ "\n")
		t.Fail()
	}
	time.Sleep(10 * time.Millisecond)
	_, err = ValidateJWT(tokenString, tokenSecret)
	if err != nil {
		return
	}
	t.Log("expected invalid token\n")
	t.Fail()
}