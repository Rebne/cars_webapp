package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path"
	"strconv"
)

// CarData contains the structure of the JSON data
type CarData struct {
	Manufacturers []Manufacturer `json:"manufacturers"`
	Categories    []Category     `json:"categories"`
	CarModels     []CarModel     `json:"carModels"`
}

// Manufacturer represents a car manufacturer
type Manufacturer struct {
	ID           int    `json:"id"`
	Name         string `json:"name"`
	Country      string `json:"country"`
	FoundingYear int    `json:"foundingYear"`
}

// Category represents a car category
type Category struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// CarModel represents a car model
type CarModel struct {
	ID             int               `json:"id"`
	Name           string            `json:"name"`
	ManufacturerID int               `json:"manufacturerId"`
	CategoryID     int               `json:"categoryId"`
	Year           int               `json:"year"`
	Specifications CarSpecifications `json:"specifications"`
	Image          string            `json:"image"`
}

// CarSpecifications represents the specifications of a car model
type CarSpecifications struct {
	Engine       string `json:"engine"`
	Horsepower   int    `json:"horsepower"`
	Transmission string `json:"transmission"`
	Drivetrain   string `json:"drivetrain"`
}

func APIHandler(w http.ResponseWriter, r *http.Request) {
	// Extract car ID from the request URL
	_, err := strconv.Atoi(r.URL.Path[len("/api/car/"):])
	if err != nil {
		http.Error(w, "Invalid car ID", http.StatusBadRequest)
		return
	}

	// Call the function to get car details from the API
	carDetails, err := getCarDataFromAPI()
	if err != nil {
		fmt.Println("Error getting car details from API:", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Respond with the car details as JSON
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(carDetails)
}

// GetManufacturerName returns the name of the manufacturer based on the given ID
func GetManufacturerName(manufacturerID int, carData CarData) string {
	for _, m := range carData.Manufacturers {
		if m.ID == manufacturerID {
			return m.Name
		}
	}
	return "Unknown Manufacturer"
}

// GetCategoryName returns the name of the category based on the given ID
func GetCategoryName(categoryID int, carData CarData) string {
	for _, c := range carData.Categories {
		if c.ID == categoryID {
			return c.Name
		}
	}
	return "Unknown Category"
}

func main() {
	// Set up a simple web server
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Call the function to get car data from the API
		carData, err := getCarDataFromAPI()
		if err != nil {
			fmt.Println("Error getting car data from API:", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Render HTML template with car data
		renderTemplate(w, carData)
	})

	// Serve static files (HTML, CSS, JavaScript) from the current directory
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	fmt.Println()
	// Handle API requests
	http.HandleFunc("/api/car/", APIHandler)

	// Start the web server on port 8080
	fmt.Println("Server is running on http://localhost:8080")
	http.ListenAndServe(":8080", nil)
}

// getCarDataFromAPI reads the car data from the data.json file
func getCarDataFromAPI() (CarData, error) {
	// Read the car data from the data.json file
	dataJSON, err := os.ReadFile("api/data.json")
	if err != nil {
		return CarData{}, err
	}

	// Create a struct to unmarshal the JSON response
	var carData CarData
	err = json.Unmarshal(dataJSON, &carData)
	if err != nil {
		return CarData{}, err
	}

	return carData, nil
}

// renderTemplate renders an HTML template with car data
func renderTemplate(w http.ResponseWriter, data CarData) {
	// Get the current directory
	dir, err := os.Getwd()
	if err != nil {
		fmt.Println("Error getting current directory:", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Construct the path to the HTML template file
	templatePath := path.Join(dir, "templates", "form.html")

	// Parse the HTML template
	tmpl, err := template.New("form.html").Funcs(template.FuncMap{
		"GetManufacturerName": GetManufacturerName,
		"GetCategoryName":     GetCategoryName,
	}).ParseFiles(templatePath)

	if err != nil {
		fmt.Println("Error parsing template:", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Execute the template with car data
	err = tmpl.Execute(w, struct {
		CarData CarData
	}{
		CarData: data,
	})
	if err != nil {
		fmt.Println("Error executing template:", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}
