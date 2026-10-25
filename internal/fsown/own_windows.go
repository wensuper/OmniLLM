package fsown

import "os"

// Mine: Windows keeps these files in the user's own profile.
func Mine(os.FileInfo) bool { return true }

// MineOrRoot: as Mine.
func MineOrRoot(os.FileInfo) bool { return true }
