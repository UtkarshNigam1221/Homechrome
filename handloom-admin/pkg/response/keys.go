package response

// Standard JSON payload keys returned from handler responses. Shared so
// admin + store handlers emit identical envelope shapes for clients.
const (
	KeyMessage = "message"
	KeyStatus  = "status"
	KeyError   = "error"
)
