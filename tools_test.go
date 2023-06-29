package toolkit

import (
	"fmt"
	"image"
	"image/png"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
)

func TestTools_RandomString(t *testing.T){
	var testTools Tools
	s:=testTools.RandomString(10)
	if len(s) != 10{
		t.Error("Wrong length of returned random string")
	}
}

var uploadTests = []struct{
	name string
	allowedTypes []string
	renameFile bool
	errorExpected bool
}{
	{name: "allowed no rename", allowedTypes: []string{"image/jpeg", "image/png", "image/gif",}, renameFile: false, errorExpected: false,},{name: "allowed rename", allowedTypes: []string{"image/jpeg", "image/png",}, renameFile: true, errorExpected: false,}, {name: "not allowed", allowedTypes: []string{"image/jpeg",  }, renameFile: false, errorExpected: true,},
	
}

func TestTools_UploadFiles(t *testing.T){
	for _, e := range uploadTests{
		pr, pw := io.Pipe()
		writer := multipart.NewWriter(pw)
		wg := sync.WaitGroup{}
		wg.Add(1)
		go func(){
			defer writer.Close()
			defer wg.Done()
		
			part, err := writer.CreateFormFile("file", "./testdata/img.png")
			if err !=nil{
				t.Error(err)
			}
			file, err := os.Open("./testdata/img.png")
			if err !=nil{
				t.Error(err)
			}
			defer file.Close()
			img, _, err := image.Decode(file)
			if err!=nil{
				t.Error("error decoding image", err)
			}
			err = png.Encode(part,img)
			if err!=nil{
				t.Error(err)
			}

		}()
		//read from the pipe which receives data
		req := httptest.NewRequest("POST", "/", pr)
		req.Header.Add("Content-Type", writer.FormDataContentType())

		var testTools Tools
		testTools.AllowedFileTypes = e.allowedTypes
		uploadedFiles, err := testTools.UploadFiles(req, "./testdata/uploads/", e.renameFile)
		if err !=nil && !e.errorExpected{
			t.Error(err)
		}
		if !e.errorExpected{
			if _, err := os.Stat(fmt.Sprintf("./testdata/uploads/%s", uploadedFiles[0].NewFilename)); os.IsNotExist(err){
				t.Errorf("%s: expected file to exists: %s", e.name, err.Error())
			}
			//clean up
			_= os.Remove(fmt.Sprintf("./testdata/uploads/%s", uploadedFiles[0].NewFilename))
		}
		if !e.errorExpected && err !=nil{
			t.Errorf("%s: errpr expected but none received", e.name)
		}
		wg.Wait()

	}
}


func TestTools_UploadOneFile(t *testing.T){
	pr, pw := io.Pipe()
	writer := multipart.NewWriter(pw)
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func(){
		defer writer.Close()
		defer wg.Done()
	
		part, err := writer.CreateFormFile("file", "./testdata/img.png")
		if err !=nil{
			t.Error(err)
		}
		file, err := os.Open("./testdata/img.png")
		if err !=nil{
			t.Error(err)
		}
		defer file.Close()
		img, _, err := image.Decode(file)
		if err!=nil{
			t.Error("error decoding image", err)
		}
		err = png.Encode(part,img)
		if err!=nil{
			t.Error(err)
		}

	}()
	//read from the pipe which receives data
	req := httptest.NewRequest("POST", "/", pr)
	req.Header.Add("Content-Type", writer.FormDataContentType())

	var testTools Tools
	
	uploadedFile, err := testTools.UploadOneFile(req, "./testdata/uploads/", true)
	if err !=nil{
		t.Error(err)
	}

		if _, err := os.Stat(fmt.Sprintf("./testdata/uploads/%s", uploadedFile.NewFilename)); os.IsNotExist(err){
			t.Errorf("expected file to exists: %s", err.Error())
		}
		//clean up
		_= os.Remove(fmt.Sprintf("./testdata/uploads/%s", uploadedFile.NewFilename))

}

func TestTools_CreateDirIfNotExist(t *testing.T){
	var testTool Tools

	err := testTool.CreateDirIfNotExist("./testdata/testDir")
	if err != nil{
		t.Error()
	}

	err = testTool.CreateDirIfNotExist("./testdata/testDir")
	if err!=nil{
		t.Error((err))
	}
	//cleanup
	_ = os.Remove("./testdata/testDir")
}

var slugTests = []struct{
	name string
	s string
	expected string
	errorExpected bool
}{
	{name:  "valid string", s: "lets slug", expected: "lets-slug", errorExpected: false },{name:  "empty string", s: "", expected: "", errorExpected: true },{name:  "complex string", s: "LET'S CODE 123!", expected: "let-s-code-123", errorExpected: false },
	{name:  "not english string", s: "ሰላም ልዑል", expected: "", errorExpected: true }, {name:  "some not english characters string", s: "!helloሰላም ልዑል", expected: "hello", errorExpected: false },
}
func TestTools_Slugify(t *testing.T){
	var testTool Tools

	for _, slugTest := range slugTests{
		slug ,err := testTool.Slugify(slugTest.s)
		if err != nil && !slugTest.errorExpected{
			t.Errorf("%s: error received when not expected: %s", slugTest.name, err.Error())
		}
		if !slugTest.errorExpected && slug!=slugTest.expected{
			t.Errorf("%s: expected %s, but got %s", slugTest.name, slugTest.expected, slug)
		}
	}

}

func TestTools_DownloadFile(t *testing.T){
	rr := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)

	var testTool Tools
	testTool.DownloadFile(rr,req,"./testdata", "img.png", "clock.png")
	res:=rr.Result()
	defer res.Body.Close()

	if res.Header["Content-Length"][0] != "534283"{
		t.Error("Wrong content length of", res.Header["Content-Length"][0])
	}
	if res.Header["Content-Disposition"][0] != "attachment; filename=\"clock.png\""{
		t.Error("Wrong content disposition, got", res.Header["Content-Disposition"][0])
	}
	_, err:= ioutil.ReadAll(res.Body)
	if err!=nil{
		t.Error(err)
	}
}