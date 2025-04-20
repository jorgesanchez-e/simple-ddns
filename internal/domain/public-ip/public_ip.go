package publicip

import "context"

type IP struct {
	V4 *string
	V6 *string
}

type Getter interface {
	GetIp(context.Context) IP
}
