package templates

import (
	"context"
	"net/url"
)

func IsHtmx(ctx context.Context) bool {
	isHtmx, ok := ctx.Value("isHtmx").(bool)
	return ok && isHtmx
}

func CurrentUrl(ctx context.Context) *url.URL {
	result, _ := ctx.Value("currentUrl").(*url.URL)
	return result
}
