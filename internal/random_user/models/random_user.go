package models

import "time"

const (
	RANDOM_COUNTER_NAME = "random-counter"
)

type RandomUser struct {
	Persons []Person `json:"results"`
	Info    Info     `json:"info"`
}

type Person struct {
	Gender     string     `json:"gender"`
	Name       Name       `json:"name"`
	Location   Location   `json:"location"`
	Email      string     `json:"email"`
	Login      Login      `json:"login"`
	Dob        Dob        `json:"dob"`
	Registered Registered `json:"registered"`
	Phone      string     `json:"phone"`
	Cell       string     `json:"cell"`
	ID         ID         `json:"id"`
	Picture    Picture    `json:"picture"`
	Nat        string     `json:"nat"`
}

type Name struct {
	Title string `json:"title"`
	First string `json:"first"`
	Last  string `json:"last"`
}

type Street struct {
	Number int    `json:"number"`
	Name   string `json:"name"`
}

type Coordinates struct {
	Latitude  string `json:"latitude"`
	Longitude string `json:"longitude"`
}

type Timezone struct {
	Offset      string `json:"offset"`
	Description string `json:"description"`
}

type Location struct {
	Street      Street      `json:"street"`
	City        string      `json:"city"`
	State       string      `json:"state"`
	Country     string      `json:"country"`
	Postcode    int         `json:"postcode"`
	Coordinates Coordinates `json:"coordinates"`
	Timezone    Timezone    `json:"timezone"`
}

type Login struct {
	UUID     string `json:"uuid"`
	Username string `json:"username"`
	Password string `json:"password"`
	Salt     string `json:"salt"`
	Md5      string `json:"md5"`
	Sha1     string `json:"sha1"`
	Sha256   string `json:"sha256"`
}

type Dob struct {
	Date time.Time `json:"date"`
	Age  int       `json:"age"`
}
type Registered struct {
	Date time.Time `json:"date"`
	Age  int       `json:"age"`
}

type ID struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type Picture struct {
	Large     string `json:"large"`
	Medium    string `json:"medium"`
	Thumbnail string `json:"thumbnail"`
}

type Info struct {
	Seed    string `json:"seed"`
	Results int    `json:"results"`
	Page    int    `json:"page"`
	Version string `json:"version"`
}

type Response struct {
	Gender   string `json:"gender"`
	FullName string `json:"fullName"`
	Address  string `json:"address"`
	Picture  string `json:"picture"`
}
