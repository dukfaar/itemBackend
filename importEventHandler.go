package main

import (
	"encoding/json"
	"fmt"

	"github.com/dukfaar/itemBackend/item"
	"github.com/globalsign/mgo/bson"
)

type RCItemEventData struct {
	Name        string `json:"name"`
	NamespaceID string `json:"namespace"`
	//Add other vars here
}

func setModelFromRCEvent(itemModel *item.Model, data RCItemEventData) {
	itemModel.Name = data.Name
	itemModel.NamespaceID = bson.ObjectIdHex(data.NamespaceID)
	//Add other vars here
}

func CreateRCEventImporter(itemService item.Service) func(msg []byte) error {
	return func(msg []byte) error {
		var itemData RCItemEventData
		err := json.Unmarshal(msg, &itemData)

		if err != nil {
			fmt.Printf("Error(%v) unmarshaling event data: %v\n", err, string(msg))
			return err
		}

		if itemData.Name == "" {
			fmt.Printf("Cant import an item without a name\n")
			return nil
		}

		itemModel, err := itemService.FindByName(itemData.Name)

		if err != nil {
			if err.Error() == "not found" {

				var newItemModel = item.Model{}
				setModelFromRCEvent(&newItemModel, itemData)

				_, err := itemService.Create(&newItemModel)

				if err != nil {
					fmt.Printf("Error(%v) saving new item: %v\n", err, newItemModel)
					return err
				}
				return nil
			} else {
				fmt.Printf("Unknown error: %v\n", err)
				return err
			}
		}

		if itemModel == nil {
			fmt.Println("Model not found")
		} else {
			setModelFromRCEvent(itemModel, itemData)

			_, err := itemService.Update(itemModel.ID.Hex(), &itemModel)

			if err != nil {
				fmt.Printf("Error(%v) updating item: %v\n", err, itemModel)
			}
		}

		return nil
	}
}
