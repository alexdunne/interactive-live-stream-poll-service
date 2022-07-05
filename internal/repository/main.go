package repository

import "fmt"

func buildPollDatabaseKey(id string) string {
	return fmt.Sprintf("POLL#%s", id)
}

func buildUserDatabaseKey(id string) string {
	return fmt.Sprintf("USER#%s", id)
}
