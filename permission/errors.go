package permission

type MissingError struct {
	Permission string
}

func (e MissingError) Error() string { return "Missing permission: " + e.Permission }

type NotLoggedIn struct {
}

func (e NotLoggedIn) Error() string { return "You are not logged in" }

type InvalidAuthHeader struct {
}

func (e InvalidAuthHeader) Error() string { return "The authorization header is invalid" }
