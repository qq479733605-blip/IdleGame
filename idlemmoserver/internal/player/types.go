package player

import "errors"

var (
	ErrInvalidItemCount   = errors.New("invalid item count")
	ErrItemNotEquippable  = errors.New("item cannot be equipped")
	ErrSequenceNotFound   = errors.New("sequence not found")
	ErrSubProjectLocked   = errors.New("sub project not unlocked")
	ErrSubProjectNotFound = errors.New("sub project not found")
)
