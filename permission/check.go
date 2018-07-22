package permission

import (
	"context"
	"strings"
)

func Check(ctx context.Context, permission string) error {
	permissionService := ctx.Value("permissionService").(*Service)
	authentication := ctx.Value("Authentication").(string)

	if authentication == "" {
		return NotLoggedIn{}
	}

	splitAuthentication := strings.Split(authentication, " ")

	if len(splitAuthentication) != 2 {
		return InvalidAuthHeader{}
	}

	if !permissionService.TokenHasPermission(splitAuthentication[1], "item.namespaceId.read") {
		return MissingError{"item"}
	}

	return nil
}
