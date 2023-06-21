package db

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
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

var Words = []DictionaryStruct{}

func JsonOpen(library string, words []DictionaryStruct) {
	//  открываем файл
	jsonFile, err := os.Open(library)
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

	err = json.Unmarshal(jsonData, &words)
	if err != nil {
		fmt.Println("Ошибка десериализации:", err)
		return
	}

}
