package models

type Credentials struct {
    Username string `json:"username" validate:"required,alphanum,min=3,max=30"`
    Password string `json:"password" validate:"required,min=6"`
}
