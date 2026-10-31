//go:build !android && !ios

package tz

import "errors"

// Desktop Go reads the local zone itself.
func systemZone() (string, error) { return "", errors.New("tz: not needed here") }
