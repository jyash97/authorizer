package resolvers

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/enum"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/session"
	"github.com/authorizerdev/authorizer/server/utils"
	"golang.org/x/crypto/bcrypt"
)

func Login(ctx context.Context, params model.LoginInput) (*model.AuthResponse, error) {
	gc, err := utils.GinContextFromContext(ctx)
	var res *model.AuthResponse
	if err != nil {
		return res, err
	}

	if constants.DISABLE_BASIC_AUTHENTICATION {
		return res, fmt.Errorf(`basic authentication is disabled for this instance`)
	}

	params.Email = strings.ToLower(params.Email)
	user, err := db.Mgr.GetUserByEmail(params.Email)
	if err != nil {
		return res, fmt.Errorf(`user with this email not found`)
	}

	if !strings.Contains(user.SignupMethods, enum.BasicAuth.String()) {
		return res, fmt.Errorf(`user has not signed up email & password`)
	}

	if user.EmailVerifiedAt == nil {
		return res, fmt.Errorf(`email not verified`)
	}

	err = bcrypt.CompareHashAndPassword([]byte(*user.Password), []byte(params.Password))

	if err != nil {
		log.Println("compare password error:", err)
		return res, fmt.Errorf(`invalid password`)
	}
	roles := constants.DEFAULT_ROLES
	currentRoles := strings.Split(user.Roles, ",")
	if len(params.Roles) > 0 {
		if !utils.IsValidRoles(currentRoles, params.Roles) {
			return res, fmt.Errorf(`invalid roles`)
		}

		roles = params.Roles
	}
	refreshToken, _, _ := utils.CreateAuthToken(user, enum.RefreshToken, roles)

	accessToken, expiresAt, _ := utils.CreateAuthToken(user, enum.AccessToken, roles)

	session.SetToken(user.ID, accessToken, refreshToken)
	utils.CreateSession(user.ID, gc)

	res = &model.AuthResponse{
		Message:     `Logged in successfully`,
		AccessToken: &accessToken,
		ExpiresAt:   &expiresAt,
		User:        utils.GetResponseUserData(user),
	}

	utils.SetCookie(gc, accessToken)

	return res, nil
}
