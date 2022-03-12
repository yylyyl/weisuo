package serverhelper

func StaticKeyAuthenticator(key string) func(string, string) bool {
	return func(remoteIp, auth string) bool {
		return auth == key
	}
}
