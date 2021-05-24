package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/mux"
	"github.com/lithammer/shortuuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var adminsCollection *mongo.Collection
var studentsCollection *mongo.Collection
var examsCollection *mongo.Collection

type Admin struct {
	ADMIN_USERNAME string
	ADMIN_PASSWORD string
	ADMIN_EMAIL    string
	ADMIN_TOKEN    string
}

type Exam struct {
	EXAM_NAME               string
	EXAM_DESC               string
	EXAM_ID                 string
	EXAM_STARTTIME          int64
	EXAM_QUESTION_BANK      []int
	EXAM_CREATOR            string
	EXAM_DURATION           int64
	EXAM_CREATION_TIMESTAMP int64
}

type Question struct {
	QUESTION_ID     string
	QUESTION_TITLE  string
	QUESTION_A      string
	QUESTION_B      string
	QUESTION_C      string
	QUESTION_D      string
	QUESTION_ANSWER string
	QUESTION_SCORE  int
}

type Student struct {
	STUDENT_USERNAME string
	STUDENT_PASSWORD string
	STUDENT_NAME     string
	STUDENT_CLASSES  []string
}

func main() {
	//connect to MONGODB
	clientOptions := options.Client().ApplyURI("mongodb://localhost:27017")
	client, err := mongo.Connect(context.TODO(), clientOptions)
	if err != nil {
		log.Fatal(err)
	}
	err = client.Ping(context.TODO(), nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Connected to MongoDB!")

	//DATABASE COLLECTIONS
	adminsCollection = client.Database("testino").Collection("admins")
	studentsCollection = client.Database("testino").Collection("students")
	examsCollection = client.Database("testino").Collection("exams")

	// Handlers
	r := mux.NewRouter()
	r.HandleFunc("/ping", PingHandler).Methods("GET")
	r.HandleFunc("/login", LoginHandler).Methods("POST")
	r.HandleFunc("/register", RegisterHandler).Methods("POST")
	r.HandleFunc("/getExams", GetExamsHandler).Methods("POST")
	r.HandleFunc("/addExam", AddNewExamHandler).Methods("POST")

	http.Handle("/", r)
	fmt.Println("listening on port 5000")
	http.ListenAndServe(":5000", nil)
}

func PingHandler(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]string{"status": "pong"})

	dt := time.Now().Format("01-02-2006 15:04:05")
	fmt.Print("\n", r.RequestURI+" "+r.Method+" "+dt, " ==> pong")
}

func LoginHandler(w http.ResponseWriter, r *http.Request) {

	dt := time.Now().Format("01-02-2006 15:04:05")
	fmt.Print("\n", r.RequestURI+" "+r.Method+" "+dt, " ==> ")

	userType := r.FormValue("userType")
	username := r.FormValue("username")
	password := r.FormValue("password")

	if userType == "admin" {
		filter := bson.D{{Key: "admin_username", Value: username},
			{Key: "admin_password", Value: password}}
		var result Admin

		err := adminsCollection.FindOne(context.TODO(), filter).Decode(&result)
		if err != nil {
			fmt.Print("Invalid login data: ", username)
			http.Error(w, "نام کاربری یا رمز عبور اشتباه است.", http.StatusBadRequest)

		} else {
			json.NewEncoder(w).Encode(map[string]string{"token": result.ADMIN_TOKEN})
			fmt.Print("logged in: ", username)
		}

	} else {

	}
}

func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	dt := time.Now().Format("01-02-2006 15:04:05")
	fmt.Print("\n", r.RequestURI+" "+r.Method+" "+dt, " ==> ")

	email := r.FormValue("email")
	username := r.FormValue("username")
	password := r.FormValue("password")

	filter := bson.D{{Key: "admin_email", Value: email}}
	var result Admin

	err := adminsCollection.FindOne(context.TODO(), filter).Decode(&result)
	if err == nil {
		http.Error(w, "این پست الکترونیک قبل استفاده شده است.", http.StatusBadRequest)
		fmt.Print("used email: " + email)

	} else {
		var result Admin
		filter := bson.D{{Key: "admin_username", Value: username}}
		err := adminsCollection.FindOne(context.TODO(), filter).Decode(&result)
		if err == nil {
			http.Error(w, "این نام کاربری قبل استفاده شده است.", http.StatusBadRequest)
			fmt.Print("used username: " + username)

		} else {
			//REGISTER

			//generat token
			type customClaims struct {
				Username string `json:username`
				jwt.StandardClaims
			}
			claims := customClaims{
				Username: username,
				StandardClaims: jwt.StandardClaims{
					ExpiresAt: 15000,
					Issuer:    "testino",
				},
			}
			token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

			signedToken, _ := token.SignedString([]byte("testino"))

			insertUser := Admin{username, password, email, signedToken}

			insertResult, err := adminsCollection.InsertOne(context.TODO(), insertUser)
			if err != nil {
				log.Println(err)
				http.Error(w, err.Error(), http.StatusBadRequest)
			}
			if insertResult != nil {
				fmt.Print("New user added: ", email)
				json.NewEncoder(w).Encode(map[string]string{"token": signedToken})
			}
		}

	}

}

func AddNewExamHandler(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("name")
	desc := r.FormValue("desc")
	startTime, _ := strconv.ParseInt(r.FormValue("startTime"), 10, 0)
	duration, _ := strconv.ParseInt(r.FormValue("duration"), 10, 0)
	username := r.FormValue("creator")
	timestampOfCreation := time.Now().Unix()

	//Generate exam ID
	id := shortuuid.New()

	insertExam := Exam{name, desc, id, startTime, nil, username, duration, timestampOfCreation}

	insertResult, err := examsCollection.InsertOne(context.TODO(), insertExam)
	if err != nil {
		log.Println(err)
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
	if insertResult != nil {
		fmt.Print("New exam added: ", name)
		json.NewEncoder(w).Encode(map[string]string{"status": name + " اضافه شد."})
	} else {
		log.Println(err)
	}
}

func GetExamsHandler(w http.ResponseWriter, r *http.Request) {
	dt := time.Now().Format("01-02-2006 15:04:05")
	fmt.Print("\n", r.RequestURI+" "+r.Method+" "+dt, " ==> ")

	username := r.FormValue("creator")
	filter := bson.D{{Key: "exam_creator", Value: username}}

	findOptions := options.Find()
	findOptions.SetLimit(0)
	cur, err := examsCollection.Find(context.TODO(), filter, findOptions)
	if err != nil {
		log.Println(err)
	}

	var results []*Exam

	for cur.Next(context.TODO()) {
		var elem Exam
		err := cur.Decode(&elem)
		if err != nil {
			log.Fatal(err)
		}
		results = append(results, &elem)
	}
	if err := cur.Err(); err != nil {
		log.Fatal(err)
	}
	cur.Close(context.TODO())

	json.NewEncoder(w).Encode(results)
	fmt.Print("Returned exams of: ", username)

}
