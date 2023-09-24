package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/Golang/MyProgram/words/pkg/logger"
	"github.com/sirupsen/logrus"
)

type DictionaryStruct struct {
	Index                                           int
	WordOriginal                                    string
	WordTranslated                                  string
	WordOriginalTranscription                       string
	WordOriginalPastSimpleSingular                  string
	WordOriginalPastSimpleSingularTranscription     string
	WordOriginalPastSimplePlural                    string
	WordOriginalPastSimplePluralTranscription       string
	WordOriginalPastParticipleSingular              string
	WordOriginalPastParticipleSingularTranscription string
	WordOriginalPastParticiplePlural                string
	WordOriginalPastParticiplePluralTranscription   string
	WordOriginalSynonyms                            string
	WordOriginalPartOfSpeech                        string
	Rating                                          int
}

type SettingsLesson struct {
	Index int
}

var SettingsSession SettingsLesson

var Words = []DictionaryStruct{}
var GoogleDict = []DictionaryStruct{}

type IndexData struct {
	Index int `db:"index"`
}

var Word1 DictionaryStruct
var WordValue DictionaryStruct

var IndexWord int

type ElementWithIndex struct {
	Index   int
	Element DictionaryStruct
}

var TenWords []DictionaryStruct

// var MyLibrary string = "library/EnglishForEveryone.json"
// var MyLibrary string = "library/weeks.json"
//var MyLibrary string = "library/HF_Networking.json"

var MyLibrary string = "library/Oxford_A1.json"
// var MyLibrary string = "library/Oxford_A2.json"


// var MyLibrary string = "library/class101.json"



//var MyLibrary string = "library/weeks.json"

func main() {

	logger.LogSetupConsole()
	logrus.Printf("запуск сервера http://localhost:8080/wordAll")

	//  открываем файл
	jsonFile, err := os.Open(MyLibrary)
	if err != nil {
		fmt.Println("Ошибка открытия файла:", err)
		return
	}
	defer jsonFile.Close()

	// Читаем содержимое файла
	jsonData, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		fmt.Println("Ошибка чтения файла:", err)
		return
	}

	// Десериализуем JSON в структуру

	err = json.Unmarshal(jsonData, &Words)
	if err != nil {
		fmt.Println("Ошибка десериализации:", err)
		return
	}

	jsonFileGoogle, err := os.Open("eng-rus_Google_v2.json")
	if err != nil {
		fmt.Println("Ошибка создания файла:", err)
		return
	}
	defer jsonFile.Close()

	// Читаем содержимое файла
	jsonDataGoogle, err := ioutil.ReadAll(jsonFileGoogle)
	if err != nil {
		fmt.Println("Ошибка чтения файла:", err)
		return
	}

	// Десериализуем JSON в структуру

	err = json.Unmarshal(jsonDataGoogle, &GoogleDict)
	if err != nil {
		fmt.Println("Ошибка десериализации:", err)
		return
	}

	//  открываем файл
	jsonFileSettings, err := os.Open("Settings.json")
	if err != nil {
		fmt.Println("Ошибка создания файла:", err)
		return
	}
	defer jsonFileSettings.Close()

	// Читаем содержимое файла
	jsonDataSettings, err := ioutil.ReadAll(jsonFileSettings)
	if err != nil {
		fmt.Println("Ошибка чтения файла:", err)
		return
	}

	// Десериализуем JSON в структуру

	err = json.Unmarshal(jsonDataSettings, &SettingsSession)
	if err != nil {
		fmt.Println("Ошибка десериализации:", err)
		return
	}

	tenWords()
	// log.Println("started http.ListenAndServe localhost:8080/word")
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	http.HandleFunc("/index", index)
	http.HandleFunc("/word", word)
	http.HandleFunc("/wordOtvet", wordOtvet)
	http.HandleFunc("/wordAdd", wordAdd)
	http.HandleFunc("/wordAll", wordAll)
	http.HandleFunc("/handleIndex", handleIndex)
	http.HandleFunc("/handleEdit", handleEdit)
	http.HandleFunc("/handleAdd", handleAdd)
	http.HandleFunc("/element-info/", handleElementInfo)
	http.HandleFunc("/wordsSearch", wordsSearch)
	http.HandleFunc("/api/search", searchHandler)
	http.HandleFunc("/Settings", Settings)
	http.HandleFunc("/SettingsSave", SettingsSave)
	http.HandleFunc("/wordAddStruct", wordAddStruct)
	http.HandleFunc("/exportToChatGPTBtn", exportToChatGPTBtn)

	http.HandleFunc("/done", done)

	http.ListenAndServe(":8080", nil)

}

func index(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("template/index.html", "template/header.html", "template/footer.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	err = tmpl.Execute(w, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
func wordAll(w http.ResponseWriter, r *http.Request) {
	//  открываем файл
	jsonFile, err := os.Open(MyLibrary)
	if err != nil {
		fmt.Println("Ошибка создания файла:", err)
		return
	}
	defer jsonFile.Close()

	// Читаем содержимое файла
	jsonData, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		fmt.Println("Ошибка чтения файла:", err)
		return
	}

	// Десериализуем JSON в структуру

	err = json.Unmarshal(jsonData, &Words)
	if err != nil {
		fmt.Println("Ошибка десериализации:", err)
		return
	}

	// Сортируем список слов по значению Rating в порядке возрастания
	sort.Slice(Words, func(i, j int) bool {
		return Words[i].Rating < Words[j].Rating
	})

	tmpl, err := template.ParseFiles("template/wordAll.html", "template/header.html", "template/footer.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = tmpl.Execute(w, Words)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func word(w http.ResponseWriter, r *http.Request) {
	//  открываем файл
	jsonFileSettings, err := os.Open("Settings.json")
	if err != nil {
		fmt.Println("Ошибка создания файла:", err)
		return
	}
	defer jsonFileSettings.Close()

	// Читаем содержимое файла
	jsonDataSettings, err := ioutil.ReadAll(jsonFileSettings)
	if err != nil {
		fmt.Println("Ошибка чтения файла:", err)
		return
	}

	// Десериализуем JSON в структуру

	err = json.Unmarshal(jsonDataSettings, &SettingsSession)
	if err != nil {
		fmt.Println("Ошибка десериализации:", err)
		return
	}

	IndexWord = findMinRatingIndex(TenWords)

	var tenWordsArr []string
	for i := range TenWords {
		tenWordsArr = append(tenWordsArr, TenWords[i].WordOriginal)
	}

	logrus.Printf("%s", tenWordsArr)

	tmpl, err := template.ParseFiles("template/word.html", "template/header.html", "template/footer.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = tmpl.Execute(w, Words[IndexWord])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

}
func wordAdd(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("template/wordAdd.html", "template/header.html", "template/footer.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if r.FormValue("WordOriginal") != "" {

		WordValue.WordOriginal = r.FormValue("WordOriginal")
		WordValue.WordOriginalTranscription = r.FormValue("WordOriginalTranscription")
		WordValue.WordTranslated = r.FormValue("WordTranslated")
		WordValue.WordOriginalPartOfSpeech = r.FormValue("WordOriginalPartOfSpeech")
		WordValue.WordOriginalSynonyms = r.FormValue("WordOriginalSynonyms")
		WordValue.WordOriginalPastSimpleSingular = r.FormValue("WordOriginalPastSimpleSingular")
		WordValue.WordOriginalPastSimpleSingularTranscription = r.FormValue("WordOriginalPastSimpleSingularTranscription")
		WordValue.WordOriginalPastSimplePlural = r.FormValue("WordOriginalPastSimplePlural")
		WordValue.WordOriginalPastSimplePluralTranscription = r.FormValue("WordOriginalPastSimplePluralTranscription")
		WordValue.WordOriginalPastParticipleSingular = r.FormValue("WordOriginalPastParticipleSingular")
		WordValue.WordOriginalPastParticipleSingularTranscription = r.FormValue("WordOriginalPastParticipleSingularTranscription")
		WordValue.WordOriginalPastParticiplePlural = r.FormValue("WordOriginalPastParticiplePlural")
		WordValue.WordOriginalPastParticiplePluralTranscription = r.FormValue("WordOriginalPastParticiplePluralTranscription")

		wordExists := false
		for _, words := range Words {
			if words.WordOriginal == WordValue.WordOriginal {
				wordExists = true
				break
			}
		}

		if !wordExists {
			logrus.Printf("Добавил слово: %s - %s", WordValue.WordOriginal, WordValue.WordTranslated)
			Words = append(Words, WordValue)

			// ... (работа с файлом и запись JSON)

			// Открываем файл для записи
			jsonFile, err := os.OpenFile(MyLibrary, os.O_WRONLY|os.O_TRUNC, 0644)
			if err != nil {
				fmt.Println("Ошибка открытия файла:", err)
				return
			}
			defer jsonFile.Close()

			// Сериализуем структуру в JSON
			jsonData, err := json.MarshalIndent(Words, "", "  ")
			if err != nil {
				fmt.Println("Ошибка сериализации:", err)
				return
			}
			// Записываем JSON в файл
			_, err = jsonFile.Write(jsonData)
			if err != nil {
				fmt.Println("Ошибка записи в файл:", err)
				return
			}
		}

	}

	err = tmpl.Execute(w, GoogleDict)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func wordOtvet(w http.ResponseWriter, r *http.Request) {
	WordValue.WordOriginal = r.FormValue("word")

	if strings.EqualFold(WordValue.WordOriginal, Words[IndexWord].WordOriginal) {
		Words[IndexWord].Rating += 1

		logrus.Printf("%s: %v", Words[IndexWord].WordOriginal, Words[IndexWord].Rating)

		// Открываем файл для записи
		jsonFile, err := os.OpenFile(MyLibrary, os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			fmt.Println("Ошибка создания файла:", err)
			return
		}
		defer jsonFile.Close()
		// Сериализуем структуру в JSON
		jsonData, err := json.MarshalIndent(Words, "", "  ")
		if err != nil {
			fmt.Println("Ошибка сериализации:", err)
			return
		}
		// Записываем JSON в файл
		_, err = jsonFile.Write(jsonData)
		if err != nil {
			fmt.Println("Ошибка записи в файл:", err)
			return
		}

		tmpl, err := template.ParseFiles("template/wordOk.html", "template/header.html", "template/footer.html")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		err = tmpl.Execute(w, Words[IndexWord])
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}

	} else {
		Words[IndexWord].Rating -= 1

		logrus.Printf("%s: %v", Words[IndexWord].WordOriginal, Words[IndexWord].Rating)

		// Открываем файл для записи
		jsonFile, err := os.OpenFile(MyLibrary, os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			fmt.Println("Ошибка создания файла:", err)
			return
		}
		defer jsonFile.Close()
		// Сериализуем структуру в JSON
		jsonData, err := json.MarshalIndent(Words, "", "  ")
		if err != nil {
			fmt.Println("Ошибка сериализации:", err)
			return
		}
		// Записываем JSON в файл
		_, err = jsonFile.Write(jsonData)
		if err != nil {
			fmt.Println("Ошибка записи в файл:", err)
			return
		}
		tmpl, err := template.ParseFiles("template/wordNot.html", "template/header.html", "template/footer.html")
		err = tmpl.Execute(w, Words[IndexWord])
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}

	}
	var CountDone int
	for _, word := range Words {
		if word.Rating > 100 {
			CountDone++
		}
	}
	logrus.Printf("Done: %v", CountDone)
}
func done(w http.ResponseWriter, r *http.Request) {
	Words[IndexWord].Rating += 100

	logrus.Printf("%s: %v", Words[IndexWord].WordOriginal, Words[IndexWord].Rating)

	// Открываем файл для записи
	jsonFile, err := os.OpenFile(MyLibrary, os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		fmt.Println("Ошибка создания файла:", err)
		return
	}
	defer jsonFile.Close()
	// Сериализуем структуру в JSON
	jsonData, err := json.MarshalIndent(Words, "", "  ")
	if err != nil {
		fmt.Println("Ошибка сериализации:", err)
		return
	}
	// Записываем JSON в файл
	_, err = jsonFile.Write(jsonData)
	if err != nil {
		fmt.Println("Ошибка записи в файл:", err)
		return
	}

	IndexWord = findMinRatingIndex(TenWords)

	tmpl, err := template.ParseFiles("template/word.html", "template/header.html", "template/footer.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = tmpl.Execute(w, Words[IndexWord])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func findMinRatingIndex(words []DictionaryStruct) int {
	minIndex := 0
	minValue := words[0].Rating

	for i, word := range words {
		if word.Rating < minValue {
			minValue = word.Rating
			minIndex = i
		}
	}

	return minIndex
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusInternalServerError)
		return
	}

	var indexData IndexData
	err = json.Unmarshal(body, &indexData)
	if err != nil {
		http.Error(w, "Error unmarshalling JSON", http.StatusBadRequest)
		return
	}

	wordsDelete(indexData.Index)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))

}

func wordsDelete(index int) {
	fmt.Println("Вызвана функция с индексом:", index)
	// Реализуйте вашу логику здесь
	Words = removeElementByIndex(Words, index)
	// Открываем файл для записи
	jsonFile, err := os.OpenFile(MyLibrary, os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		fmt.Println("Ошибка открытия файла:", err)
		return
	}
	defer jsonFile.Close()
	// Сериализуем структуру в JSON
	jsonData, err := json.MarshalIndent(Words, "", "  ")
	if err != nil {
		fmt.Println("Ошибка сериализации:", err)
		return
	}
	// Записываем JSON в файл
	_, err = jsonFile.Write(jsonData)
	if err != nil {
		fmt.Println("Ошибка записи в файл:", err)
		return
	}

}
func removeElementByIndex(words []DictionaryStruct, index int) []DictionaryStruct {
	if index < 0 || index >= len(words) {
		return words
	}
	return append(words[:index], words[index+1:]...)
}

func handleEdit(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var requestData struct {
		Index                                           int    `db:"index"`
		WordOriginal                                    string `db:"WordOriginal"`
		WordTranslated                                  string `db:"WordTranslated"`
		WordOriginalTranscription                       string `db:"WordOriginalTranscription"`
		WordOriginalPastSimpleSingular                  string `db:"WordOriginalPastSimpleSingular"`
		WordOriginalPastSimpleSingularTranscription     string `db:"WordOriginalPastSimpleSingularTranscription"`
		WordOriginalPastSimplePlural                    string `db:"WordOriginalPastSimplePlural"`
		WordOriginalPastSimplePluralTranscription       string `db:"WordOriginalPastSimplePluralTranscription"`
		WordOriginalPastParticipleSingular              string `db:"WordOriginalPastParticipleSingular"`
		WordOriginalPastParticipleSingularTranscription string `db:"WordOriginalPastParticipleSingularTranscription"`
		WordOriginalPastParticiplePlural                string `db:"WordOriginalPastParticiplePlural"`
		WordOriginalPastParticiplePluralTranscription   string `db:"WordOriginalPastParticiplePluralTranscription"`
		WordOriginalSynonyms                            string `db:"WordOriginalSynonyms"`
		// WordOriginalPartOfSpeech                        string `db:"WordOriginalPartOfSpeech"`
		Rating int `db:"Rating"`
	}

	err := json.NewDecoder(r.Body).Decode(&requestData)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	index := requestData.Index
	if index < 0 || index >= len(Words) {
		http.Error(w, "Invalid index value", http.StatusBadRequest)
		return
	}

	// Обновление элемента с новыми данными
	Words[index].WordOriginal = requestData.WordOriginal
	Words[index].WordOriginalTranscription = requestData.WordOriginalTranscription
	Words[index].WordTranslated = requestData.WordTranslated
	// Words[index].WordOriginalPartOfSpeech = requestData.WordOriginalPartOfSpeech
	Words[index].WordOriginalSynonyms = requestData.WordOriginalSynonyms
	Words[index].Rating = requestData.Rating
	Words[index].WordOriginalPastSimpleSingular = requestData.WordOriginalPastSimpleSingular
	Words[index].WordOriginalPastSimpleSingularTranscription = requestData.WordOriginalPastSimpleSingularTranscription
	Words[index].WordOriginalPastSimplePlural = requestData.WordOriginalPastSimplePlural
	Words[index].WordOriginalPastSimplePluralTranscription = requestData.WordOriginalPastSimplePluralTranscription
	Words[index].WordOriginalPastParticipleSingular = requestData.WordOriginalPastParticipleSingular
	Words[index].WordOriginalPastParticipleSingularTranscription = requestData.WordOriginalPastParticipleSingularTranscription
	Words[index].WordOriginalPastParticiplePlural = requestData.WordOriginalPastParticiplePlural
	Words[index].WordOriginalPastParticiplePluralTranscription = requestData.WordOriginalPastParticiplePluralTranscription

	// Обновление файла данных (если есть) и другие операции, если необходимо
	// Открываем файл для записи
	jsonFile, err := os.OpenFile(MyLibrary, os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		fmt.Println("Ошибка открытия файла:", err)
		return
	}
	defer jsonFile.Close()

	// Сериализуем структуру в JSON
	jsonData, err := json.MarshalIndent(Words, "", "  ")
	if err != nil {
		fmt.Println("Ошибка сериализации:", err)
		return
	}
	// Записываем JSON в файл
	_, err = jsonFile.Write(jsonData)
	if err != nil {
		fmt.Println("Ошибка записи в файл:", err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func handleAdd(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var requestData struct {
		Index                                           int    `db:"index"`
		WordOriginal                                    string `db:"WordOriginal"`
		WordTranslated                                  string `db:"WordTranslated"`
		WordOriginalTranscription                       string `db:"WordOriginalTranscription"`
		WordOriginalPastSimpleSingular                  string `db:"WordOriginalPastSimpleSingular"`
		WordOriginalPastSimpleSingularTranscription     string `db:"WordOriginalPastSimpleSingularTranscription"`
		WordOriginalPastSimplePlural                    string `db:"WordOriginalPastSimplePlural"`
		WordOriginalPastSimplePluralTranscription       string `db:"WordOriginalPastSimplePluralTranscription"`
		WordOriginalPastParticipleSingular              string `db:"WordOriginalPastParticipleSingular"`
		WordOriginalPastParticipleSingularTranscription string `db:"WordOriginalPastParticipleSingularTranscription"`
		WordOriginalPastParticiplePlural                string `db:"WordOriginalPastParticiplePlural"`
		WordOriginalPastParticiplePluralTranscription   string `db:"WordOriginalPastParticiplePluralTranscription"`
		WordOriginalSynonyms                            string `db:"WordOriginalSynonyms"`
		WordOriginalPartOfSpeech                        string `db:"WordOriginalPartOfSpeech"`
		Rating                                          int    `db:"Rating"`
	}

	err := json.NewDecoder(r.Body).Decode(&requestData)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	index := requestData.Index
	if index < 0 || index >= len(Words) {
		http.Error(w, "Invalid index value", http.StatusBadRequest)
		return
	}

	Words = append(Words, DictionaryStruct(requestData))

	// Обновление файла данных (если есть) и другие операции, если необходимо
	// Открываем файл для записи
	jsonFile, err := os.OpenFile(MyLibrary, os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		fmt.Println("Ошибка открытия файла:", err)
		return
	}
	defer jsonFile.Close()

	// Сериализуем структуру в JSON
	jsonData, err := json.MarshalIndent(Words, "", "  ")
	if err != nil {
		fmt.Println("Ошибка сериализации:", err)
		return
	}
	// Записываем JSON в файл
	_, err = jsonFile.Write(jsonData)
	if err != nil {
		fmt.Println("Ошибка записи в файл:", err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func handleElementInfo(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("template/ElementInfo.html", "template/header.html", "template/footer.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	urlPath := r.URL.Path
	indexStr := strings.TrimPrefix(urlPath, "/element-info/")
	index, err := strconv.Atoi(indexStr)
	if err != nil {
		http.Error(w, "Invalid index format", http.StatusBadRequest)
		return
	}

	if index < 0 || index >= len(Words) {
		http.Error(w, "Index out of range", http.StatusBadRequest)
		return
	}

	elementWithIndex := ElementWithIndex{
		Index:   index,
		Element: Words[index],
	}

	err = tmpl.Execute(w, elementWithIndex)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func searchHandler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q") // Измените "query" на "q"
	results := searchWords(query)

	w.Header().Set("Content-Type", "application/db")
	json.NewEncoder(w).Encode(results)
}

func searchWords(query string) []DictionaryStruct {
	results := []DictionaryStruct{}
	query = strings.ToLower(query)

	for _, word := range GoogleDict {
		wordOriginalLower := strings.ToLower(word.WordOriginal)
		wordTranslatedLower := strings.ToLower(word.WordTranslated)
		if strings.HasPrefix(wordOriginalLower, query) || strings.HasPrefix(wordTranslatedLower, query) {
			results = append(results, word)
		}
	}

	return results
}

func wordsSearch(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/db")
	json.NewEncoder(w).Encode(GoogleDict)
}

func tenWords() {
	//  открываем файл
	jsonFileSettings, err := os.Open("Settings.json")
	if err != nil {
		fmt.Println("Ошибка создания файла:", err)
		return
	}
	defer jsonFileSettings.Close()

	// Читаем содержимое файла
	jsonDataSettings, err := ioutil.ReadAll(jsonFileSettings)
	if err != nil {
		fmt.Println("Ошибка чтения файла:", err)
		return
	}

	// Десериализуем JSON в структуру

	err = json.Unmarshal(jsonDataSettings, &SettingsSession)
	if err != nil {
		fmt.Println("Ошибка десериализации:", err)
		return
	}
	TenWords = findMinRatingWords(Words, SettingsSession.Index)
}

func findMinRatingWords(words []DictionaryStruct, count int) []DictionaryStruct {
	sort.Slice(words, func(i, j int) bool {
		return words[i].Rating < words[j].Rating
	})

	if len(words) < count {
		return words
	}

	return words[:count]
}

func Settings(w http.ResponseWriter, r *http.Request) {

	tmpl, err := template.ParseFiles("template/Settings.html", "template/header.html", "template/footer.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = tmpl.Execute(w, SettingsSession)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

}

func SettingsSave(w http.ResponseWriter, r *http.Request) {
	if r.FormValue("Settings") != "" {

		strIndex := r.FormValue("Settings")
		index, err := strconv.Atoi(strIndex)
		if err != nil {
			// обработка ошибки, возможно, strIndex не может быть преобразован в int
			http.Error(w, "Invalid index value", http.StatusBadRequest)
			return
		}
		SettingsSession.Index = index

		// Открываем файл для записи
		jsonFile, err := os.OpenFile("Settings.json", os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			fmt.Println("Ошибка открытия файла:", err)
			return
		}
		defer jsonFile.Close()

		// Сериализуем структуру в JSON
		jsonData, err := json.MarshalIndent(SettingsSession, "", "  ")
		if err != nil {
			fmt.Println("Ошибка сериализации:", err)
			return
		}
		// Записываем JSON в файл
		_, err = jsonFile.Write(jsonData)
		if err != nil {
			fmt.Println("Ошибка записи в файл:", err)
			return
		}

	}

	tmpl, err := template.ParseFiles("template/Settings.html", "template/header.html", "template/footer.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = tmpl.Execute(w, SettingsSession)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	tenWords()
}
func wordAddStruct(w http.ResponseWriter, r *http.Request) {
	// Получение данных от клиента
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Ошибка чтения тела запроса", http.StatusBadRequest)
		return
	}

	var wordsArray []DictionaryStruct
	err = json.Unmarshal(body, &wordsArray)
	if err != nil {
		http.Error(w, "Ошибка декодирования JSON", http.StatusBadRequest)
		return
	}

	// Загрузка существующего JSON файла
	jsonFile := MyLibrary
	jsonData, err := ioutil.ReadFile(jsonFile)
	if err != nil {
		http.Error(w, "Ошибка чтения JSON файла", http.StatusInternalServerError)
		return
	}

	var existingWords []DictionaryStruct
	err = json.Unmarshal(jsonData, &existingWords)
	if err != nil {
		http.Error(w, "Ошибка декодирования существующего JSON", http.StatusInternalServerError)
		return
	}

	// Создание map для проверки уникальности слов
	wordOriginalMap := make(map[string]bool)
	for _, word := range existingWords {
		wordOriginalMap[word.WordOriginal] = true
	}

	// Добавление новых слов к существующим данным
	for _, newWord := range wordsArray {
		if _, exists := wordOriginalMap[newWord.WordOriginal]; !exists {
			existingWords = append(existingWords, newWord)
		}
	}

	// Обновление JSON файла
	updatedJsonData, err := json.MarshalIndent(existingWords, "", "  ")
	if err != nil {
		http.Error(w, "Ошибка кодирования JSON", http.StatusInternalServerError)
		return
	}

	err = ioutil.WriteFile(jsonFile, updatedJsonData, 0644)
	if err != nil {
		http.Error(w, "Ошибка записи в JSON файл", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Слова успешно добавлены в JSON файл!"))
}
func exportToChatGPTBtn(w http.ResponseWriter, r *http.Request) {
	//  открываем файл
	jsonFile, err := os.Open(MyLibrary)
	if err != nil {
		fmt.Println("Ошибка создания файла:", err)
		return
	}
	defer jsonFile.Close()

	// Читаем содержимое файла
	jsonData, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		fmt.Println("Ошибка чтения файла:", err)
		return
	}

	// Десериализуем JSON в структуру

	err = json.Unmarshal(jsonData, &Words)
	if err != nil {
		fmt.Println("Ошибка десериализации:", err)
		return
	}

	// Сортируем список слов по значению Rating в порядке возрастания
	sort.Slice(Words, func(i, j int) bool {
		return Words[i].Rating < Words[j].Rating
	})

	tmpl, err := template.ParseFiles("template/exportToChatGPTBtn.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = tmpl.Execute(w, Words)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
