package item

type ItemService interface {
	Create(*Model) (*Model, error)
	FindByID(string) (*Model, error)
	HasElementBeforeID(id string) (bool, error)
	HasElementAfterID(id string) (bool, error)

	Count() (int, error)

	List(first *int32, last *int32, before *string, after *string) ([]Model, error)
}
