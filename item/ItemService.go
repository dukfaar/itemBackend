package item

import (
	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type ItemService interface {
	Create(*Model) (*Model, error)
	DeleteByID(id string) (string, error)
	FindByID(string) (*Model, error)
	HasElementBeforeID(id string) (bool, error)
	HasElementAfterID(id string) (bool, error)

	Count() (int, error)

	List(first *int32, last *int32, before *string, after *string) ([]Model, error)
}

type MgoItemService struct {
	db         *mgo.Database
	collection *mgo.Collection
}

func NewMgoItemService(db *mgo.Database) *MgoItemService {
	return &MgoItemService{
		db:         db,
		collection: db.C("items"),
	}
}

func (s *MgoItemService) Create(model *Model) (*Model, error) {
	model.ID = bson.NewObjectId()

	err := s.collection.Insert(model)

	return model, err
}

func (s *MgoItemService) DeleteByID(id string) (string, error) {
	err := s.collection.RemoveId(bson.ObjectIdHex(id))

	return id, err
}

func (s *MgoItemService) FindByID(id string) (*Model, error) {
	var result Model

	err := s.collection.FindId(bson.ObjectIdHex(id)).One(&result)

	return &result, err
}

func (s *MgoItemService) HasElementBeforeID(id string) (bool, error) {
	query := bson.M{}

	query["_id"] = bson.M{
		"$lt": bson.ObjectIdHex(id),
	}

	count, err := s.collection.Find(query).Count()
	return count > 0, err
}

func (s *MgoItemService) HasElementAfterID(id string) (bool, error) {
	query := bson.M{}

	query["_id"] = bson.M{
		"$gt": bson.ObjectIdHex(id),
	}

	count, err := s.collection.Find(query).Count()
	return count > 0, err
}

func (s *MgoItemService) Count() (int, error) {
	count, err := s.collection.Find(bson.M{}).Count()
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
		count, _ := s.collection.Find(query).Count()

		limit = int(*last)
		skip = count - limit
	}

	var result []Model
	err := s.collection.Find(query).Skip(skip).Limit(limit).All(&result)
	return result, err
}
