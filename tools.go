package toolkit

import (
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const randomStringSrc = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

type Tools struct{
	MaxFileSize int
	AllowedFileTypes []string
}

func (t *Tools) RandomString(n int) string {
	s, r := make([]rune, n), []rune(randomStringSrc)
	for i := range s {
		p, _ := rand.Prime(rand.Reader, len(r))
		x,y := p.Uint64(), uint64(len(r))
		s[i] = r[x%y]

	}
	return string(s)
}

//used to save info of an uploaded file
type UploadedFile struct{
	NewFilename string
	OriginFileName string
	FileSize int64

}


func (t *Tools) UploadOneFile(r *http.Request, uploadDir string, rename ...bool)(*UploadedFile, error){
	renameFile := true
	if len(rename) > 0{
		renameFile = rename[0]
	}
	files, err := t.UploadFiles(r, uploadDir, renameFile)
	if err !=nil{
		return nil, err
	}
	return files[0], nil
}

func (t *Tools) UploadFiles(r *http.Request, uploadDir string, rename ...bool)([]*UploadedFile, error){
	renameFile := true
	if len(rename) > 0{
		renameFile = rename[0]
	}

	var uploadedFiles []*UploadedFile

	if t.MaxFileSize == 0{
		t.MaxFileSize = 1024*1024*1024
	}
	
	err := t.CreateDirIfNotExist(uploadDir)
	if err!=nil{
		return nil, err
	}
	err= r.ParseMultipartForm(int64(t.MaxFileSize))
	if err != nil {
		return nil, errors.New("The uploaded file is too big")
	}

	for _,fHeaders := range r.MultipartForm.File{
		for _, header := range fHeaders{
			uploadedFiles, err = func(uploadedFiles []*UploadedFile)([]*UploadedFile, error){
				var uploadedFile UploadedFile
				infile, err := header.Open()
				if err != nil{
					return nil, err
				}
				defer infile.Close()
				buff := make([]byte, 512)
				_,err = infile.Read(buff)
				if err != nil{
					return nil, err
				}
				
				allowed := false
				fileType := http.DetectContentType(buff)
				// allowedTypes := []string{"image/jpeg", "image/png", "image/gif"}
				if len(t.AllowedFileTypes) > 0 {
					for _, allowedType := range t.AllowedFileTypes{
						if strings.EqualFold(fileType, allowedType){
							allowed = true
					}
				}
				}else{
					allowed = true
				}
				if !allowed{
					return nil, fmt.Errorf("The uploaded file type %s is not permitted", fileType)
				}
				_, err = infile.Seek(0,0)
				if err != nil{
					return nil, err
				}
				if renameFile{
					uploadedFile.NewFilename = fmt.Sprintf("%s%s", t.RandomString(25), filepath.Ext(header.Filename))
				}else{
					uploadedFile.NewFilename = header.Filename
				}
				uploadedFile.OriginFileName = header.Filename
				var outfile *os.File
				defer outfile.Close()

				if outfile, err = os.Create(filepath.Join(uploadDir, uploadedFile.NewFilename)); err != nil{
					if os.IsNotExist(err){
						if err:= os.MkdirAll(uploadDir, os.ModePerm); err!=nil{
							return nil, err
						}
						if outfile, err = os.Create(filepath.Join(uploadDir, uploadedFile.NewFilename)); err != nil{
							return nil,err
						}
					}
				}else{
					fileSize, err := io.Copy(outfile, infile)
					if err != nil{
						return nil, err
					}
					uploadedFile.FileSize = fileSize

				}

				uploadedFiles = append(uploadedFiles, &uploadedFile)
				return uploadedFiles, nil
			}(uploadedFiles)
			if err != nil{
				return uploadedFiles, err
			}

		}

	}
	return uploadedFiles, nil
} 


func (t *Tools) CreateDirIfNotExist(path string) error{
	const mode = 0755
	if _, err := os.Stat(path); os.IsNotExist(err){
		err := os.MkdirAll(path,mode)
		if err!=nil{
			return err
		}
	} 
	return nil
}

//get a string and make it url safe
func (t *Tools) Slugify(s string) (string, error){
	if s == ""{
		return s, errors.New("Empty string is not permitted")
	}
	var re = regexp.MustCompile(`[^a-z\d]+`)
	slug:= strings.Trim(re.ReplaceAllString(strings.ToLower(s), "-"),"-")
	if len(slug) == 0 {
		return slug, errors.New("After slugify the string, slug is zero length")
	}
	return slug,nil
}