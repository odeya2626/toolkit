package toolkit

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

const randomStringSrc = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

type Tools struct{
	MaxFileSize int
	AllowedFileTypes []string
	MaxJSONSize int
	AllowUnknownFields bool

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
func (t *Tools) DownloadFile(w http.ResponseWriter, r *http.Request, p, file, displayName string){
	filePath := path.Join(p, file)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", displayName))
	http.ServeFile(w,r,filePath)


}
//type for sending JSON response
type JSONResponse struct{
	Error bool `json:"error"`
	Message string `json:"message"`
	Data interface{} `json:"data,omitempty"`

}
func (t *Tools) ReadJSON(w http.ResponseWriter, r *http.Request, data interface{}) error{
	maxBytes := 1024*1024
	if t.MaxJSONSize !=0{
		maxBytes = t.MaxJSONSize
	}
	r.Body = http.MaxBytesReader(w, r.Body, int64(maxBytes))
	dec := json.NewDecoder(r.Body)

	if(!t.AllowUnknownFields){
		dec.DisallowUnknownFields()
	}
	err:=dec.Decode(data)
	if err!=nil{
		var syntaxError *json.SyntaxError
		var unmarshalTypeError *json.UnmarshalTypeError
		var invalidUnmarshalError *json.InvalidUnmarshalError
		switch{
		case errors.As(err, &syntaxError):
			return fmt.Errorf("json contains syntax error (at character %d)", syntaxError.Offset)
		case errors.Is(err, io.ErrUnexpectedEOF):
			return errors.New("Body contains badly-formed JSON")
		case errors.As(err, &unmarshalTypeError):
			if unmarshalTypeError.Field != ""{
				return fmt.Errorf("Body contains incorrect JSON type for field %q", unmarshalTypeError.Field)
			}
			return fmt.Errorf("Body contains incorrect JSON type (at character %d)", unmarshalTypeError.Offset)
		case errors.Is(err, io.EOF):
			return errors.New("Body must not be empty")
		case strings.HasPrefix(err.Error(), "json: unknown field"):
			fileName:= strings.TrimPrefix(err.Error(), "json: unknown field")
			return fmt.Errorf("Body contains unknown key %s", fileName)
		case err.Error() == "http: request body too large":
			return fmt.Errorf("Body must not be larger than %d bytes", maxBytes)
		case errors.As(err, &invalidUnmarshalError):
			return fmt.Errorf("Error unmarshalling JSON: %s", err.Error())
		
		default:
			return err
		}
	}
	err = dec.Decode(&struct{}{})
	if err!= io.EOF{
		return errors.New("Body must contain only one json")
	}
	return nil


}