package item

import (
	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type MgoItemService struct {
	db             *mgo.Database
	itemCollection *mgo.Collection
}

func NewMgoItemService(db *mgo.Database) *MgoItemService {
	return &MgoItemService{
		db:             db,
		itemCollection: db.C("items"),
	}
}

func (s *MgoItemService) Create(model *Model) (*Model, error) {
	model.ID = bson.NewObjectId()

	err := s.itemCollection.Insert(model)

	return model, err
}

func (s *MgoItemService) DeleteByID(id string) (string, error) {
	err := s.itemCollection.RemoveId(bson.ObjectIdHex(id))

	return id, err
}

func (s *MgoItemService) FindByID(id string) (*Model, error) {
	var result Model

	err := s.itemCollection.FindId(bson.ObjectIdHex(id)).One(&result)

	return &result, err
}

func (s *MgoItemService) HasElementBeforeID(id string) (bool, error) {
	query := bson.M{}

	query["_id"] = bson.M{
		"$lt": bson.ObjectIdHex(id),
	}

	count, err := s.itemCollection.Find(query).Count()
	return count > 0, err
}

func (s *MgoItemService) HasElementAfterID(id string) (bool, error) {
	query := bson.M{}

	query["_id"] = bson.M{
		"$gt": bson.ObjectIdHex(id),
	}

	count, err := s.itemCollection.Find(query).Count()
	return count > 0, err
}

func (s *MgoItemService) Count() (int, error) {
	count, err := s.itemCollection.Find(bson.M{}).Count()
	return count, err
}

func (s *MgoItemService) List(first *int32, last *int32, before *string, after *string) ([]Model, error) {
	query := bson.M{}

	if after != nil {
		query["_id"] = bson.M{
			"$gt": bson.ObjectIdHex(*after),
		}
	}

	if before != nil {
		query["_id"] = bson.M{
			"$lt": bson.ObjectIdHex(*before),
		}
	}

	var (
		skip  int
		limit int
	)

	if first != nil {
		limit = int(*first)
	}

	if last != nil {
		count, _ := s.itemCollection.Find(query).Count()

		limit = int(*last)
		skip = count - limit
	}

	var result []Model
	err := s.itemCollection.Find(query).Skip(skip).Limit(limit).All(&result)
	return result, err
}
