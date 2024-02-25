package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"path"
	"sync"
	"time"
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

	// Start the web server on port 8080
	fmt.Println("Server is running on http://localhost:8080")
	fmt.Println("This is the test WITH goroutines")
	http.ListenAndServe(":8080", nil)
}

func getData(s string, ptr interface{}, ch chan<- error, wg *sync.WaitGroup) {

	client := http.Client{
		Timeout: time.Second * 5,
	}
	response, err := client.Get(s)
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

// getCarDataFromAPI reads the car data from the data.json file
func getCarDataFromAPI() (CarData, error) {

	start := time.Now()
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

	fmt.Println("Time it took to get data from API: ", time.Since(start))
	return CarData{
		Manufacturers: manufacturers,
		CarModels:     models,
		Categories:    categories,
	}, err
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
