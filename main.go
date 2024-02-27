package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"strconv"
	"sync"
)

type CarData struct {
	Manufacturers []Manufacturer `json:"manufacturers"`
	Categories    []Category     `json:"categories"`
	CarModels     []CarModel     `json:"carModels"`
	Message       string
}

type Manufacturer struct {
	ID           int    `json:"id"`
	Name         string `json:"name"`
	Country      string `json:"country"`
	FoundingYear int    `json:"foundingYear"`
}

type Category struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type CarModel struct {
	ID             int               `json:"id"`
	Name           string            `json:"name"`
	ManufacturerID int               `json:"manufacturerId"`
	CategoryID     int               `json:"categoryId"`
	Year           int               `json:"year"`
	Specifications CarSpecifications `json:"specifications"`
	Image          string            `json:"image"`
}

type CarSpecifications struct {
	Engine       string `json:"engine"`
	Horsepower   int    `json:"horsepower"`
	Transmission string `json:"transmission"`
	Drivetrain   string `json:"drivetrain"`
}

var templateIndex *template.Template

var compareIndex *template.Template

func init() {

	templateIndex, _ = template.New("form.html").Funcs(template.FuncMap{
		"GetManufacturerData": GetManufacturerData,
		"GetCategoryName":     GetCategoryName,
	}).ParseFiles("templates/form.html")

	compareIndex, _ = template.New("compare.html").Funcs(template.FuncMap{
		"GetManufacturerData": GetManufacturerData,
		"GetCategoryName":     GetCategoryName,
	}).ParseFiles("templates/compare.html")
}

func main() {
	port := ":8080"
	localHost := "http://localhost:8080"
	http.HandleFunc("/compare", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			renderTemplate(w, CarData{}, compareIndex)
		} else {
			http.Error(w, "", http.StatusBadRequest)
		}
		return
	})
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		carData, err := getCarDataFromAPI()
		if err != nil {
			fmt.Println("Error getting car data from API:", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		if r.Method == "POST" {
			// for testing
			fmt.Println("I received POST signal")
			if err := r.ParseForm(); err != nil {
				http.Error(w, "Failed to parse form", http.StatusBadRequest)
			}
			var count int

			for range r.Form {
				count++
			}

			if count != 2 {
				carData.Message = "You can only select 2 options"
			} else {
				http.Redirect(w, r, localHost+port+"/compare", http.StatusSeeOther)
				return
			}
		}
		renderTemplate(w, carData, templateIndex)

	})

	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	fmt.Println("Server is running on ", localHost)
	http.ListenAndServe(port, nil)
}

func getData(s string, ptr interface{}, ch chan<- error, wg *sync.WaitGroup) {

	response, err := http.Get(s)
	if err != nil {
		fmt.Printf("Failed to fetch data: %v\n", err)
		ch <- err
		wg.Done()
		return
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		fmt.Printf("Failed to read response body: %v\n", err)
		ch <- err
		wg.Done()
		return
	}

	if err := json.Unmarshal(body, ptr); err != nil {
		fmt.Printf("Failed to unmarshal JSON: %v\n", err)
		ch <- err
		wg.Done()
		return
	}

	wg.Done()
}

func getCarDataFromAPI() (CarData, error) {

	modelsEndpoint := "http://localhost:3000/api/models"
	manufacturerEndpoint := "http://localhost:3000/api/manufacturers"
	categoriesEndpoint := "http://localhost:3000/api/categories"

	var manufacturers []Manufacturer
	var models []CarModel
	var categories []Category
	wg := &sync.WaitGroup{}
	errch := make(chan error)

	wg.Add(3)

	go getData(manufacturerEndpoint, &manufacturers, errch, wg)

	go getData(modelsEndpoint, &models, errch, wg)

	go getData(categoriesEndpoint, &categories, errch, wg)

	wg.Wait()

	close(errch)

	var err error

	for e := range errch {
		if e != nil {
			err = e
			break
		}
	}
	if err != nil {
		return CarData{}, err
	}
	return CarData{
		Manufacturers: manufacturers,
		CarModels:     models,
		Categories:    categories,
	}, err
}

func renderTemplate(w http.ResponseWriter, data CarData, tmpl *template.Template) {

	err := tmpl.Execute(w, struct {
		CarData CarData
	}{
		CarData: data,
	})
	if err != nil {
		fmt.Println("Error executing template:", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func GetManufacturerData(manufacturerID int, carData CarData, detailType string) string {
	for _, manufacturer := range carData.Manufacturers {
		if manufacturer.ID == manufacturerID {
			switch detailType {
			case "Country":
				return manufacturer.Country
			case "Name":
				return manufacturer.Name
			case "FoundingYear":
				return strconv.Itoa(manufacturer.FoundingYear)
			}
		}
	}
	return ""
}

func GetCategoryName(categoryID int, carData CarData) string {
	for _, c := range carData.Categories {
		if c.ID == categoryID {
			return c.Name
		}
	}
	return "Unknown Category"
}
