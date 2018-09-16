package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
)

func FetchXivdbItemData(ID int32) ([]byte, error) {
	idString := strconv.FormatInt(int64(ID), 10)

	itemResponse, err := http.Get("https://api.xivdb.com/item/" + idString)

	if err != nil {
		fmt.Errorf("Error getting item: %v", err)
		return nil, err
	}
	defer itemResponse.Body.Close()

	result, err := ioutil.ReadAll(itemResponse.Body)

	if err != nil {
		fmt.Errorf("Error reading item: %v", err)
		return nil, err
	}

	return result, nil
}

type XivdbItemListResponse struct {
	ID int32 `json:"id"`
}
