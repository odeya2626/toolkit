package toolkit

import (
	"fmt"
	"image"
	"image/png"
	"io"
	"mime/multipart"
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