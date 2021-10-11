package sbz

import (
	"github.com/subiz/header"
)

const VERSION = "4.0"

type Config struct {
	AccountId string
	ApiKey    string
	ApiURL    string
}

var cf *Config

// explicit pass credential
func Init(accid, apikey, apiurl string) {
	if apiurl == "" {
		apiurl = "https://api.subiz.com.vn/" + VERSION
	}
	cf = &Config{AccountId: accid, ApiKey: apikey, ApiURL: apiurl}
}

// Update
// update("usaccid", &header.User{
//
func UpdateUser(userid string, u *header.User) error {
	_, err := RequestHttp("POST", "/users/"+userid, u, nil, 0)
	return err
}
