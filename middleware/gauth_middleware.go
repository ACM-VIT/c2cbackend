package middleware

import (
	"c2cbackend/initializer"
	"c2cbackend/models"
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

var (
	once        sync.Once
	provider    *oidc.Provider
	defaultAud  string
	allowedAuds map[string]struct{}
	initErr     error
)

func initOIDC() error {
	once.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Discover Google OIDC config (JWKS, endpoints, etc.)
		provider, initErr = oidc.NewProvider(ctx, "https://accounts.google.com")
		if initErr != nil {
			return
		}

    // Preferred server-side env var
    defaultAud = strings.TrimSpace(os.Getenv("GOOGLE_OAUTH_CLIENT_ID"))
    // Common fallbacks used in some deployments
    if defaultAud == "" {
        defaultAud = strings.TrimSpace(os.Getenv("GOOGLE_CLIENT_ID"))
    }
    if defaultAud == "" {
        defaultAud = strings.TrimSpace(os.Getenv("NEXT_PUBLIC_GOOGLE_CLIENT_ID"))
    }
		allowedAuds = make(map[string]struct{})

		// Prefer an explicit allow-list if provided (supports multiple client IDs).
		if raw := os.Getenv("GOOGLE_OAUTH_ALLOWED_AUDS"); raw != "" {
			for _, a := range strings.Split(raw, ",") {
				aud := strings.TrimSpace(a)
				if aud != "" {
					allowedAuds[aud] = struct{}{}
				}
			}
		}

		// If only single client ID is configured, use that.
        if len(allowedAuds) == 0 && defaultAud == "" {
            initErr = errors.New("missing GOOGLE_OAUTH_CLIENT_ID/GOOGLE_CLIENT_ID (or GOOGLE_OAUTH_ALLOWED_AUDS)")
            return
        }
		if defaultAud != "" {
			allowedAuds[defaultAud] = struct{}{}
		}
	})
	return initErr
}

func verifyGoogleIDToken(ctx context.Context, raw string) (*oidc.IDToken, map[string]any, error) {
	if err := initOIDC(); err != nil {
		return nil, nil, err
	}

	verifier := provider.Verifier(&oidc.Config{
		SkipClientIDCheck: true,
	})

	idToken, err := verifier.Verify(ctx, raw)
	if err != nil {
		return nil, nil, err
	}

	var claims map[string]any
	if err := idToken.Claims(&claims); err != nil {
		return nil, nil, err
	}

	switch aud := claims["aud"].(type) {
	case string:
		if _, ok := allowedAuds[aud]; !ok {
			return nil, nil, errors.New("unauthorized audience")
		}
	case []any:
		ok := false
		for _, v := range aud {
			if s, sOk := v.(string); sOk {
				if _, ok = allowedAuds[s]; ok {
					break
				}
			}
		}
		if !ok {
			return nil, nil, errors.New("unauthorized audience")
		}
	default:
		return nil, nil, errors.New("invalid audience claim")
	}

	if idToken.Issuer != "https://accounts.google.com" {
		return nil, nil, errors.New("invalid issuer")
	}

	return idToken, claims, nil
}

func GoogleClaims() fiber.Handler {
    return func(c *fiber.Ctx) error {
        authHeader := c.Get("Authorization")
        if authHeader == "" {
            return c.Next()
        }
        parts := strings.SplitN(authHeader, " ", 2)
        if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" || parts[1] == "" {
            return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
                "error": "Invalid Authorization header format",
            })
        }

        idToken, claims, err := verifyGoogleIDToken(context.Background(), parts[1])
        if err != nil {
            return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
                "error": "Invalid or expired token",
            })
        }

		if sub, ok := claims["sub"].(string); ok && sub != "" {
			c.Locals("uid", sub)
		}
		c.Locals("claims", claims)
		_ = idToken
		return c.Next()
	}
}

func GoogleAuth() fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
				"error": "Missing Authorization header",
			})
		}
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" || parts[1] == "" {
			return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid Authorization header format",
			})
		}
		raw := parts[1]

		idToken, claims, err := verifyGoogleIDToken(context.Background(), raw)
		if err != nil {
			return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid or expired token",
			})
		}

		// Require email + verified
		email, _ := claims["email"].(string)
		emailVerified, _ := claims["email_verified"].(bool)
		if email == "" {
			return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
				"error": "Email not present in token claims",
			})
		}
		if !emailVerified {
			return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
				"error": "Email is not verified",
			})
		}

		var user models.User
		if err := initializer.Database.Db.Where("email = ?", email).First(&user).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return c.Status(http.StatusNotFound).JSON(fiber.Map{
					"error": "User not found",
				})
			}
			log.Println("DB error:", err)
			return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to retrieve user",
			})
		}

		c.Locals("user", user)
		if sub, ok := claims["sub"].(string); ok {
			c.Locals("uid", sub)
		} else {
			c.Locals("uid", "")
		}

		c.Locals("claims", map[string]any{
			"email":          email,
			"email_verified": emailVerified,
			"sub":            claims["sub"],
			"name":           claims["name"],
			"picture":        claims["picture"],
			"hd":             claims["hd"],
			"iss":            idToken.Issuer,
			"aud":            claims["aud"],
			"exp":            idToken.Expiry.Unix(),
		})

		return c.Next()
	}
}
