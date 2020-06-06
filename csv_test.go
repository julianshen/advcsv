package advcsv

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"testing"

	assert "github.com/stretchr/testify/assert"
)

const csvStr = `標題,內容,更多內容,評價,標籤
"Mark Zuckerberg tries to explain his inaction on Trump posts to outraged staff","Facebook (FB) CEO Mark Zuckerberg sought on Tuesday to ease employee outrage over his inaction on incendiary remarks recently posted by President Donald Trump.","During a company-wide town hall, Zuckerberg struggled to explain his decision-making process as many of his employees","2","facebook,trump"
"Dwayne Johnson makes powerful plea for leadership","Dwayne "The Rock" Johnson posted a video calling out President Donald Trump in the wake of George Floyd's death.","","5","trump"
`

var (
	expectedResult []Result
)

type Tags []string

func (tags *Tags) UnmarshalCSV(data string) error {
	*tags = append(*tags, strings.Split(data, ",")...)
	return nil
}

type Rating int

func (rating *Rating) UnmarshalCSV(data string) error {
	fmt.Println("AAs:" + data)
	iRating, err := strconv.Atoi(data)

	if err == nil {
		*rating = Rating(iRating)
		return nil
	}

	return err
}

type Result struct {
	Title   string  `csv:"標題"`
	Content string  `csv:"內容"`
	Rating  *Rating `csv:"評價"`
	Tags    *Tags   `csv:"標籤"`
}

func init() {
	rating2 := Rating(2)
	rating5 := Rating(5)
	expectedResult = []Result{
		Result{
			"Mark Zuckerberg tries to explain his inaction on Trump posts to outraged staff",
			"Facebook (FB) CEO Mark Zuckerberg sought on Tuesday to ease employee outrage over his inaction on incendiary remarks recently posted by President Donald Trump.",
			&rating2,
			&Tags{
				"facebook",
				"trump",
			},
		},
		Result{
			"Dwayne Johnson makes powerful plea for leadership",
			"Dwayne \"The Rock\" Johnson posted a video calling out President Donald Trump in the wake of George Floyd's death.",
			&rating5,
			&Tags{
				"trump",
			},
		},
	}
}

func TestValidateType(t *testing.T) {
	// Spec: Support pointer to array of struct
	//       Support pointer to array of pointer to struct
	var result1 []Result
	err := validateType(&result1)

	assert.Nil(t, err, "Should support pointer to array of struct")

	var result2 []*Result
	err = validateType(&result2)

	assert.Nil(t, err, "Should support pointer to array of pointer to struct")

	err = validateType(result1)
	assert.EqualValues(t, &UnsupportedTypeError{
		reflect.TypeOf(result1),
	}, err, "Should not support this kind of type")
}

func TestConstructCsvFields(t *testing.T) {
	headers := []string{
		"標題",
		"內容",
		"更多內容",
		"評價",
		"標籤",
	}

	var r Result
	csvFields := constructCsvFields(headers, reflect.TypeOf(r))

	assert.Len(t, csvFields, 4, "Should only have 4 fields matched")

	assert.True(t, func() bool {
		expected := map[string]int{
			"標題": 0,
			"內容": 1,
			"評價": 3,
			"標籤": 4,
		}
		for _, csvField := range csvFields {
			if index, ok := expected[csvField.headTitle]; !ok {
				return false
			} else if csvField.index != index {
				return false
			}
		}
		return true
	}(), "Should match all fields")
}

func TestUnmarshalRecord(t *testing.T) {
	headers := []string{
		"標題",
		"內容",
		"更多內容",
		"評價",
		"標籤",
	}
	var r Result
	csvFields := constructCsvFields(headers, reflect.TypeOf(r))
	record := []string{"Title 1", "Content", "more", "2", "1a,23,s"}
	res, _ := unmarshalRecord(record, reflect.TypeOf(r), csvFields)
	var rating Rating
	rating = 2
	fmt.Println(res.Interface())
	assert.EqualValues(t, Result{
		"Title 1", "Content", &rating, &Tags{"1a", "23", "s"},
	}, res.Interface())

	ptrRes, _ := unmarshalRecord(record, reflect.TypeOf(&r), csvFields)
	assert.EqualValues(t, &Result{
		"Title 1", "Content", &rating, &Tags{"1a", "23", "s"},
	}, ptrRes.Interface())
}

func TestUnmarshal(t *testing.T) {
	var r []Result
	csvReader := strings.NewReader(csvStr)
	err := Unmarshal(csvReader, &r)
	if err != nil {
		t.Error(err)
	}
	assert.EqualValues(t, expectedResult, r)
}
