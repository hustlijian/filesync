package comm

type (
	Request struct {
		Cmd  string `json:"cmd"`
		Name string `json:"name"`
		Data []byte `json:"data"`
	}
)

const (
	CheckToken = "checktoken"
	ReqFiles   = "reqfiles"
	WriteFile  = "writefile"
	DeleteFile = "deleteFile"
)
