package main

import (
	"fmt"
	"github.com/civet148/log"
	"net/url"
)

func ParseUrl(strUrl string) (scheme string, host string) {
	uri, err := url.Parse(strUrl)
	if err != nil {
		log.Panic("parse url %s error [%s]", strUrl, err.Error())
	}
	return uri.Scheme, uri.Host
}

func BuildListenUrl(scheme string, port uint32) string {
	return fmt.Sprintf("%s://0.0.0.0:%v", scheme, port)
}
