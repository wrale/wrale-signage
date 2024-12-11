package content

import "errors"

var (
	ErrContentStale      = errors.New("content not seen recently")
	ErrContentUnreliable = errors.New("content has high error rate")
)