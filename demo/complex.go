package demo

import nstd "github.com/foomo/gotsrpc/demo/nested"

type Address struct {
	City              string   `json:"city,omitempty"`
	Signs             []string `json:"signs,omitempty"`
	SecretServerCrap  bool     `json:"-"`
	PeoplePtr         []*Person
	ArrayOfMaps       []map[string]bool
	ArrayArrayAddress [][]*Address
	People            []Person
	MapCrap           map[string]map[int]bool
	NestedPtr         *nstd.Nested
	NestedStruct      nstd.Nested
}

type Person struct {
	Name          string
	AddressPtr    *Address `json:"address"`
	AddressStruct Address
	Addresses     map[string]*Address

	InlinePtr *struct {
		Foo bool
	}
	InlineStruct struct {
		Bar string
	}
	iAmPrivate string
}

func (s *Service) ExtractAddress(person *Person) (addr *Address, e *Err) {
	if person.AddressPtr != nil {
		return person.AddressPtr, nil
	}
	return nil, &Err{"there is no address on that person"}
}

func (s *Service) TestScalarInPlace() ScalarInPlace {
	return ScalarInPlace("hier")
}

func (s *Service) Nest() *nstd.Nested {
	return nil
}

func (s *Service) GiveMeAScalar() (amount nstd.Amount, wahr nstd.True, hier ScalarInPlace) {
	//func (s *Service) giveMeAScalar() (amount nstd.Amount, wahr nstd.True, hier ScalarInPlace) {
	return nstd.Amount(10), nstd.ItIsTrue, ScalarInPlace("hier")
}
