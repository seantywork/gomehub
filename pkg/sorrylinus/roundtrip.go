package sorrylinus

import (
	"fmt"

	"github.com/seantywork/gomehub/pkg/com"
)

func RoundTrip(u string, req *com.RT_REQ_DATA) (*com.RT_RESP_DATA, error) {

	var resp com.RT_RESP_DATA

	c, okay := SOLIREG[u]

	if !okay {

		return nil, fmt.Errorf("rt: no such user: %s", u)

	}

	err := c.WriteJSON(req)

	if err != nil {
		return nil, fmt.Errorf("rt: write: %s", err.Error())
	}

	err = c.ReadJSON(&resp)

	if err != nil {

		return nil, fmt.Errorf("rt: read: %s", err.Error())

	}

	return &resp, nil
}
