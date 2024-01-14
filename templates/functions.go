package templates

import "context"

func IsHtmx(ctx context.Context) bool {
	isHtmx, ok := ctx.Value("isHtmx").(bool)
	return ok && isHtmx
}
