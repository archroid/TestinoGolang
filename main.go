package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
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
var questionBankCollection *mongo.Collection
var questionsCollection *mongo.Collection

type Admin struct {
	ADMIN_USERNAME    string
	ADMIN_NAME        string
	ADMIN_PROFILE_URL string
	ADMIN_PASSWORD    string
	ADMIN_EMAIL       string
	ADMIN_TOKEN       string
}

type Exam struct {
	EXAM_NAME             string
	EXAM_DESC             string
	EXAM_ID               string
	EXAM_STARTTIME        int64
	EXAM_QUESTION_BANK_ID string
	EXAM_CREATOR          string
	EXAM_DURATION         int64
	EXAM_ICON_URL         string
}

type Question struct {
	QUESTION_ID      string
	QUESTION_TITLE   string
	QUESTION_A       string
	QUESTION_B       string
	QUESTION_C       string
	QUESTION_D       string
	QUESTION_ANSWER  string
	QUESTION_BANK_ID string
}

type QuestionBank struct {
	QUESTION_BANK_ID      string
	QUESTION_BANK_CREATOR string
	QUESTION_BANK_NAME    string
}

type Student struct {
	STUDENT_USERNAME    string
	STUDENT_NAME        string
	STUDENT_PROFILE_URL string
	STUDENT_PASSWORD    string
	STUDENT_TOKEN       string
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
	questionBankCollection = client.Database("testino").Collection("question_bank")
	questionsCollection = client.Database("testino").Collection("questions")

	// Handlers
	r := mux.NewRouter()
	r.HandleFunc("/ping", PingHandler).Methods("GET")
	r.HandleFunc("/login", LoginHandler).Methods("POST")
	r.HandleFunc("/register", RegisterHandler).Methods("POST")

	r.HandleFunc("/getUser", GetUserHandler).Methods("POST")

	r.HandleFunc("/getQuestionBank", getQuestionBankHandler).Methods("POST")
	r.HandleFunc("/getQuestionBanks", getQuestionBanksHandler).Methods("POST")
	r.HandleFunc("/addQuestionBank", AddQuestionBankHandler).Methods("POST")
	go r.HandleFunc("/addQuestion", AddQuestionHandler).Methods("POST")
	r.HandleFunc("/getQuestions", GetQuestionsHandler).Methods("POST")

	r.HandleFunc("/getExam", GetExamHandler).Methods("POST")
	r.HandleFunc("/getExams", GetExamsHandler).Methods("POST")
	r.HandleFunc("/addExam", AddNewExamHandler).Methods("POST")
	r.HandleFunc("/deleteExam", DeleteExamHandler).Methods("POST")

	r.HandleFunc("/uploadImage", UploadImageHandler).Methods("POST")

	staticDir := "/images/"
	http.Handle(staticDir, http.StripPrefix(staticDir, http.FileServer(http.Dir("."+staticDir))))

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
			fmt.Print("Invalid login data: ", username+"\n")
			http.Error(w, "نام کاربری یا رمز عبور اشتباه است.", http.StatusBadRequest)

		} else {
			json.NewEncoder(w).Encode(map[string]string{"token": result.ADMIN_TOKEN})
			fmt.Print("logged in: ", username+"\n")
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
			fmt.Print("used username: " + username + "\n")

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

			insertUser := Admin{username, username, "default", password, email, signedToken}

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
	questionBankId := r.FormValue("questionBankId")

	//Generate exam ID
	id := shortuuid.New()

	insertExam := Exam{name, desc, id, startTime, questionBankId, username, duration, "default"}

	insertResult, err := examsCollection.InsertOne(context.TODO(), insertExam)
	if err != nil {
		log.Println(err)
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
	if insertResult != nil {
		fmt.Print("New exam added: ", name)
		json.NewEncoder(w).Encode(map[string]string{"id": id})
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

func AddQuestionBankHandler(w http.ResponseWriter, r *http.Request) {
	dt := time.Now().Format("01-02-2006 15:04:05")
	fmt.Print("\n", r.RequestURI+" "+r.Method+" "+dt, " ==> ")

	username := r.FormValue("creator")
	name := r.FormValue("name")

	//Generate exam ID
	id := shortuuid.New()

	insertQuestionBank := QuestionBank{id, username, name}

	insertResult, err := questionBankCollection.InsertOne(context.TODO(), insertQuestionBank)
	if err != nil {
		log.Println(err)
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
	if insertResult != nil {
		fmt.Print("New questionBank added: ", name+"\n")
		json.NewEncoder(w).Encode(map[string]string{"id": id})
	} else {
		log.Println(err)
	}

}

func AddQuestionHandler(w http.ResponseWriter, r *http.Request) {
	dt := time.Now().Format("01-02-2006 15:04:05")
	fmt.Print("\n", r.RequestURI+" "+r.Method+" "+dt, " ==> ")

	title := r.FormValue("title")
	A := r.FormValue("A")
	B := r.FormValue("B")
	C := r.FormValue("C")
	D := r.FormValue("D")
	answer := r.FormValue("answer")
	bankId := r.FormValue("bankId")
	id := shortuuid.New()

	insertQuestion := Question{id, title, A, B, C, D, answer, bankId}

	insertResult, err := questionsCollection.InsertOne(context.TODO(), insertQuestion)
	if err != nil {
		log.Println(err)
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
	if insertResult != nil {
		fmt.Print("New question added: ", id+"\n")
		json.NewEncoder(w).Encode(map[string]string{"id": id})
	} else {
		log.Println(err)
	}
}

func GetExamHandler(w http.ResponseWriter, r *http.Request) {
	dt := time.Now().Format("01-02-2006 15:04:05")
	fmt.Print("\n", r.RequestURI+" "+r.Method+" "+dt, " ==> ")

	id := r.FormValue("id")

	filter := bson.D{{Key: "exam_id", Value: id}}
	var result Exam

	err := examsCollection.FindOne(context.TODO(), filter).Decode(&result)
	if err != nil {
		fmt.Print("No exam found: ", id+"\n")
		http.Error(w, "صفحه مورد نظر یافت نشد", http.StatusBadRequest)

	} else {
		json.NewEncoder(w).Encode(result)
		fmt.Print("Found exam: ", result.EXAM_NAME+"\n")
	}

}

func DeleteExamHandler(w http.ResponseWriter, r *http.Request) {
	dt := time.Now().Format("01-02-2006 15:04:05")
	fmt.Print("\n", r.RequestURI+" "+r.Method+" "+dt, " ==> ")

	id := r.FormValue("id")
	filter := bson.D{{Key: "exam_id", Value: id}}

	deleteResult, err := examsCollection.DeleteMany(context.TODO(), filter)
	if err != nil {
		log.Fatal(err)
		fmt.Print("No exam found: ", id+"\n")
		http.Error(w, "not found", http.StatusBadRequest)
	}

	fmt.Printf("Deleted %v documents in the trainers collection\n", deleteResult.DeletedCount)
	json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})

}

func GetQuestionsHandler(w http.ResponseWriter, r *http.Request) {
	dt := time.Now().Format("01-02-2006 15:04:05")
	fmt.Print("\n", r.RequestURI+" "+r.Method+" "+dt, " ==> ")

	id := r.FormValue("id")

	filter := bson.D{{Key: "question_bank_id", Value: id}}

	findOptions := options.Find()
	findOptions.SetLimit(0)
	cur, err := questionsCollection.Find(context.TODO(), filter, findOptions)
	if err != nil {
		log.Println(err)
	}

	var results []*Question

	for cur.Next(context.TODO()) {
		var elem Question
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
	fmt.Print("Returned Questions of: ", id)

}

func getQuestionBanksHandler(w http.ResponseWriter, r *http.Request) {
	dt := time.Now().Format("01-02-2006 15:04:05")
	fmt.Print("\n", r.RequestURI+" "+r.Method+" "+dt, " ==> ")

	username := r.FormValue("creator")

	filter := bson.D{{Key: "question_bank_creator", Value: username}}

	findOptions := options.Find()
	findOptions.SetLimit(0)
	cur, err := questionBankCollection.Find(context.TODO(), filter, findOptions)
	if err != nil {
		log.Println(err)
	}

	var results []*QuestionBank

	for cur.Next(context.TODO()) {
		var elem QuestionBank
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
	fmt.Print("Returned QuestionBanks of: ", username)

}

func getQuestionBankHandler(w http.ResponseWriter, r *http.Request) {
	dt := time.Now().Format("01-02-2006 15:04:05")
	fmt.Print("\n", r.RequestURI+" "+r.Method+" "+dt, " ==> ")

	id := r.FormValue("id")

	filter := bson.D{{Key: "question_bank_id", Value: id}}

	var result QuestionBank

	err := questionBankCollection.FindOne(context.TODO(), filter).Decode(result)
	if err != nil {
		if err != nil {
			fmt.Print("No questionBank found: ", id+"\n")
			http.Error(w, "صفحه مورد نظر یافت نشد", http.StatusBadRequest)

		} else {
			json.NewEncoder(w).Encode(result)
			fmt.Print("Found questionBank: ", result.QUESTION_BANK_NAME+"\n")
		}
	}
}

func UploadImageHandler(w http.ResponseWriter, r *http.Request) {
	dt := time.Now().Format("01-02-2006 15:04:05")
	fmt.Print("\n", r.RequestURI+" "+r.Method+" "+dt, " ==> ")

	file, handler, err := r.FormFile("file")

	fileName := r.FormValue("fileName")
	filePath := r.FormValue("filePath")

	if err != nil {
		panic(err)
	}
	defer file.Close()

	os.MkdirAll("images/studentProfiles", os.ModePerm)
	os.MkdirAll("images/adminProfiles", os.ModePerm)
	os.MkdirAll("images/examIcons", os.ModePerm)

	f, err := os.OpenFile(filePath+"/"+fileName, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	_, _ = io.Copy(f, file)

	switch filePath {
	case "images/studentProfiles":
		filter := bson.D{{"student_username", fileName}}

		update := bson.D{
			{"$set", bson.D{
				{"student_profile_url", "http://192.168.1.108:5000/" + filePath + "/" + fileName},
			}},
		}

		_, err := studentsCollection.UpdateOne(context.TODO(), filter, update)
		if err != nil {
			log.Print(err)
		}
		break
	case "images/adminProfiles":
		filter := bson.D{{"admin_username", fileName}}

		update := bson.D{
			{"$set", bson.D{
				{"admin_profile_url", "http://192.168.1.108:5000/" + filePath + "/" + fileName},
			}},
		}

		_, err := adminsCollection.UpdateOne(context.TODO(), filter, update)
		if err != nil {
			log.Print(err)
		}
		break
	case "images/examIcons":
		filter := bson.D{{"exam_id", fileName}}

		update := bson.D{
			{"$set", bson.D{
				{"exam_icon_url", "http://192.168.1.108:5000/" + filePath + "/" + fileName},
			}},
		}

		_, err := examsCollection.UpdateOne(context.TODO(), filter, update)
		if err != nil {
			log.Print(err)
		}
		break
	}

	fmt.Print(filePath + handler.Filename)

	json.NewEncoder(w).Encode(map[string]string{"status": "http://192.168.1.108:5000/" + filePath + "/" + fileName})
}

func GetUserHandler(w http.ResponseWriter, r *http.Request) {

	dt := time.Now().Format("01-02-2006 15:04:05")
	fmt.Print("\n", r.RequestURI+" "+r.Method+" "+dt, " ==> ")

	username := r.FormValue("username")
	userType := r.FormValue("userType")

	switch userType {
	case "admin":

		filter := bson.D{{Key: "admin_username", Value: username}}
		var result Admin

		err := adminsCollection.FindOne(context.TODO(), filter).Decode(&result)
		if err != nil {
			fmt.Print("Return Error: ", username+"\n")
			http.Error(w, "Error", http.StatusBadRequest)

		} else {
			json.NewEncoder(w).Encode(result)
			fmt.Print("Return user: ", username+"\n")
		}
		break
	}

}
