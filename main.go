package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
)

type CarData struct {
	Manufacturers []Manufacturer `json:"manufacturers"`
	Categories    []Category     `json:"categories"`
	CarModels     []CarModel     `json:"carModels"`
	Message       string
	IsPopup       bool
	CompareModels []CarModel
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

func init() {

	templateIndex, _ = template.New("form.html").Funcs(template.FuncMap{
		"GetManufacturerData": GetManufacturerData,
		"GetCategoryName":     GetCategoryName,
		"CompareHorsepower":   CompareHorsepower,
	}).ParseFiles("templates/form.html")

}

func main() {
	port := ":8080"
	localHost := "http://localhost"

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		carData, err := getCarDataFromAPI()
		if err != nil {
			fmt.Println("Error getting car data from API:", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-type", "text/html")

		carData.IsPopup = false

		if r.Method == "POST" {
			if err := r.ParseForm(); err != nil {
				http.Error(w, "Failed to parse form", http.StatusBadRequest)
			}
			var count int

			for range r.Form["option"] {
				count++
			}
			if count != 2 {
				carData.Message = "You have to select 2 options"
			} else {
				carData.IsPopup = true
				for _, carID := range r.Form["option"] {
					carData.CompareModels = append(carData.CompareModels, getCarModel(carID, &carData))
				}
				renderTemplate(w, carData, templateIndex)
				return
			}
		}

		renderTemplate(w, carData, templateIndex)

	})

	// Handle the "/filtered" route
	http.HandleFunc("/filtered", filteredHandler)

	// Serve static files (HTML, CSS, JavaScript) from the current directory
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	fmt.Println("Server is running on ", localHost+port)
	http.ListenAndServe(port, nil)
}

func CompareHorsepower(a int, b int) bool {
	if a == b {
		return false
	}

	return a > b
}
func getCarModel(id string, carData *CarData) CarModel {
	for _, car := range carData.CarModels {
		target, _ := strconv.Atoi(id)
		if car.ID == target {
			return car
		}
	}
	return CarModel{}
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
func filteredHandler(w http.ResponseWriter, r *http.Request) {
	// Parse the form data from the request
	err := r.ParseForm()
	if err != nil {
		fmt.Println("Error parsing form:", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Access the form values
	manufacturer := r.Form.Get("manufacturer")
	category := r.Form.Get("category")
	drivetrain := r.Form.Get("drivetrain")
	transmission := r.Form.Get("transmission")
	horsepower := r.Form.Get("horsepower")

	// Call the function to get car data from the API
	carData, err := getCarDataFromAPI()
	if err != nil {
		fmt.Println("Error getting car data from API:", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Call the function to get filtered car data from the API
	filteredCarData, err := getFilteredCarDataFromAPI(manufacturer, category, drivetrain, transmission, horsepower, carData)
	if err != nil {
		fmt.Println("Error getting filtered car data from API:", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Render HTML template with filtered car data
	renderTemplate(w, filteredCarData, templateIndex)
}

func getFilteredCarDataFromAPI(manufacturer, category, drivetrain, transmission, horsepower string, carData CarData) (CarData, error) {
	var filteredCarData CarData

	if horsepower == "All" {
		return carData, nil
	}

	// Parse the horsepower range
	minHP, maxHP, err := parseHorsepowerRange(horsepower)
	if err != nil {
		return CarData{}, err
	}

	// Iterate through car models and apply filters
	for _, carModel := range carData.CarModels {
		// Check if the manufacturer filter is set and match
		if manufacturer != "" && GetManufacturerData(carModel.ManufacturerID, carData, "Name") != manufacturer {
			continue
		}

		// Check if the category filter is set and match
		if category != "" && GetCategoryName(carModel.CategoryID, carData) != category {
			continue
		}

		// Check if the drivetrain filter is set and match
		if drivetrain != "" && carModel.Specifications.Drivetrain != drivetrain {
			continue
		}

		// Check if the transmission filter is set and match
		if transmission != "" {
			// Categorize transmissions into broader categories
			transmissionCategory := categorizeTransmission(carModel.Specifications.Transmission)

			// Check if the categorized transmission matches the filter
			if transmissionCategory != transmission {
				continue
			}
		}

		// Check if the horsepower filter is set and match
		if minHP != nil && maxHP != nil {
			if carModel.Specifications.Horsepower < *minHP || carModel.Specifications.Horsepower > *maxHP {
				continue
			}
		}

		filteredCarData.CarModels = append(filteredCarData.CarModels, carModel)
	}

	filteredCarData.Manufacturers = carData.Manufacturers
	filteredCarData.Categories = carData.Categories

	return filteredCarData, nil
}

func parseHorsepowerRange(horsepowerRange string) (*int, *int, error) {

	// Split the range string into min and max
	parts := strings.Split(horsepowerRange, "-")
	if len(parts) != 2 {
		return nil, nil, fmt.Errorf("invalid horsepower range format")
	}

	// Parse min and max values
	minHP, err := strconv.Atoi(parts[0])
	if err != nil {
		return nil, nil, fmt.Errorf("invalid minimum horsepower value")
	}

	maxHP, err := strconv.Atoi(parts[1])
	if err != nil {
		return nil, nil, fmt.Errorf("invalid maximum horsepower value")
	}

	return &minHP, &maxHP, nil
}

func categorizeTransmission(originalTransmission string) string {
	// Convert the transmission to lowercase for case-insensitive matching
	lowerTransmission := strings.ToLower(originalTransmission)

	// Check for specific transmission types
	if strings.Contains(lowerTransmission, "manual") {
		return "Manual"
	} else if strings.Contains(lowerTransmission, "automatic") {
		return "Automatic"
	} else if strings.Contains(lowerTransmission, "cvt") {
		return "CVT"
	}
	// Default to the original transmission value if not matched
	return originalTransmission
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
