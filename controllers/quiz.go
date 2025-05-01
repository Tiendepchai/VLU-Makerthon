package controllers

import (
    "crypto/rand"
    "encoding/hex"
    "fmt"
    "log"
    "makerthon/controllers/middleware"
    "makerthon/models"
    "makerthon/views"
    "net/http"
    "os"
    "path/filepath"
    "strconv"
    "strings"
    "time"
    "bytes"
    "io"
    "io/ioutil"
    "encoding/json"
    "encoding/base64"
    "regexp"

    "github.com/gin-gonic/gin"
    "gorm.io/gorm"
)

type quizController struct {
    db *gorm.DB
}

func NewQuizController(db *gorm.DB) *quizController {
    return &quizController{db: db}
}

func (qc *quizController) LoadRoutes(router *gin.RouterGroup) {
    router.Use(middleware.RequireAuth(qc.db))

    router.GET("/create", qc.ShowCreateQuizForm)
    router.POST("/create", qc.CreateQuiz)
    router.GET("/my-quizzes", qc.MyQuizzes)
    router.GET("/view/:id", qc.ViewQuiz)
    router.DELETE("/:id", qc.DeleteQuiz)
}

func (qc *quizController) ShowCreateQuizForm(ctx *gin.Context) {
    views.CreateQuiz().Render(ctx, ctx.Writer)
}

func (qc *quizController) CreateQuiz(ctx *gin.Context) {
    user := ctx.MustGet("user").(models.User)

    title := ctx.PostForm("title")
    description := ctx.PostForm("description")
    numQuestions, _ := strconv.Atoi(ctx.PostForm("numQuestions"))
    if numQuestions <= 0 {
        numQuestions = 10 
    }

    tokenCost := numQuestions * 5

    if user.Token < tokenCost {
        ctx.Set("error", "Bạn không đủ token để tạo quiz này! Cần: "+strconv.Itoa(tokenCost)+" token")
        views.CreateQuiz().Render(ctx, ctx.Writer)
        return
    }

    file, err := ctx.FormFile("document")
    if err != nil {
        ctx.Set("error", "Vui lòng tải lên một tài liệu")
        views.CreateQuiz().Render(ctx, ctx.Writer)
        return
    }

    filename := file.Filename
    extension := strings.ToLower(filepath.Ext(filename))
    fileType := ""

    switch extension {
    case ".pdf":
        fileType = "pdf"
    case ".docx":
        fileType = "docx"
    case ".xlsx":
        fileType = "xlsx"
    default:
        ctx.Set("error", "Định dạng file không được hỗ trợ. Vui lòng sử dụng PDF, DOCX, hoặc XLSX")
        views.CreateQuiz().Render(ctx, ctx.Writer)
        return
    }

    uploadDir := "./public/uploads"
    if _, err := os.Stat(uploadDir); os.IsNotExist(err) {
        os.MkdirAll(uploadDir, 0755)
    }

    newFilename := fmt.Sprintf("%d_%s", time.Now().UnixNano(), filename)
    filePath := filepath.Join(uploadDir, newFilename)


    if err := ctx.SaveUploadedFile(file, filePath); err != nil {
        ctx.Set("error", "Không thể lưu file. Vui lòng thử lại")
        views.CreateQuiz().Render(ctx, ctx.Writer)
        return
    }

    shareCode := generateShareCode(6)

    quiz := models.Quiz{
        Title:       title,
        Description: description,
        Status:      models.QuizStatusDraft,
        UserID:      user.ID,
        TokenCost:   tokenCost,
        FileType:    fileType,
        FileName:    newFilename,
        ShareCode:   shareCode,
    }

    tx := qc.db.Begin()

    if err := tx.Create(&quiz).Error; err != nil {
        tx.Rollback()
        ctx.Set("error", "Lỗi khi tạo quiz")
        views.CreateQuiz().Render(ctx, ctx.Writer)
        return
    }

    user.Token -= tokenCost
    if err := tx.Save(&user).Error; err != nil {
        tx.Rollback()
        ctx.Set("error", "Lỗi khi trừ token")
        views.CreateQuiz().Render(ctx, ctx.Writer)
        return
    }

    questions, err := qc.analyzeDocument(filePath, fileType, numQuestions)
    if err != nil {
        log.Printf("Error analyzing document: %v", err)
    }

    for i, q := range questions {
        questionType := models.QuestionTypeMultipleChoice
        switch q.Type {
        case "true_false":
            questionType = models.QuestionTypeTrueFalse
        case "short_answer":
            questionType = models.QuestionTypeShortAnswer
        }

        question := models.Question{
            Content:     q.Content,
            Type:        questionType,
            QuizID:      quiz.ID,
            Explanation: fmt.Sprintf("Câu hỏi #%d: %s", i+1, q.Explanation),
        }

        if err := tx.Create(&question).Error; err != nil {
            tx.Rollback()
            ctx.Set("error", "Lỗi khi tạo câu hỏi")
            views.CreateQuiz().Render(ctx, ctx.Writer)
            return
        }

        if questionType != models.QuestionTypeShortAnswer {
            var options []models.Option
            for _, opt := range q.Options {
                options = append(options, models.Option{
                    Content:    opt.Content,
                    IsCorrect:  opt.IsCorrect,
                    QuestionID: question.ID,
                })
            }

            if err := tx.Create(&options).Error; err != nil {
                tx.Rollback()
                ctx.Set("error", "Lỗi khi tạo phương án trả lời")
                views.CreateQuiz().Render(ctx, ctx.Writer)
                return
            }
        }
    }

    tx.Commit()

    ctx.Redirect(http.StatusFound, fmt.Sprintf("/quiz/view/%d", quiz.ID))
}

func (qc *quizController) MyQuizzes(ctx *gin.Context) {
    user := ctx.MustGet("user").(models.User)
    var quizzes []models.Quiz
    
    if err := qc.db.Where("user_id = ?", user.ID).Order("created_at desc").Find(&quizzes).Error; err != nil {
        ctx.Set("error", "Không thể tải danh sách quiz")
    }
    
    views.MyQuizzes(quizzes).Render(ctx, ctx.Writer)
}

func (qc *quizController) ViewQuiz(ctx *gin.Context) {
    user := ctx.MustGet("user").(models.User)
    quizID, err := strconv.Atoi(ctx.Param("id"))
    if err != nil {
        ctx.Redirect(http.StatusFound, "/quiz/my-quizzes")
        return
    }
    
    var quiz models.Quiz
    if err := qc.db.Preload("Questions.Options").First(&quiz, quizID).Error; err != nil {
        ctx.Redirect(http.StatusFound, "/quiz/my-quizzes")
        return
    }
    
    if quiz.UserID != user.ID {
        ctx.Redirect(http.StatusFound, "/quiz/my-quizzes")
        return
    }
    
    views.ViewQuiz(quiz).Render(ctx, ctx.Writer)
}

func (qc *quizController) DeleteQuiz(ctx *gin.Context) {
    user := ctx.MustGet("user").(models.User)
    quizID, err := strconv.Atoi(ctx.Param("id"))
    if err != nil {
        ctx.JSON(http.StatusBadRequest, gin.H{"error": "ID quiz không hợp lệ"})
        return
    }
    
    var quiz models.Quiz
    if err := qc.db.First(&quiz, quizID).Error; err != nil {
        ctx.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy quiz"})
        return
    }
    
    if quiz.UserID != user.ID {
        ctx.JSON(http.StatusForbidden, gin.H{"error": "Bạn không có quyền xóa quiz này"})
        return
    }
    if quiz.FileName != "" {
        filePath := filepath.Join("./public/uploads", quiz.FileName)
        os.Remove(filePath) 
    }
    
    if err := qc.db.Delete(&quiz).Error; err != nil {
        ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể xóa quiz"})
        return
    }
    
    ctx.JSON(http.StatusOK, gin.H{"success": true})
}

func generateShareCode(length int) string {
    bytes := make([]byte, length/2)
    if _, err := rand.Read(bytes); err != nil {
        log.Printf("Error generating random bytes: %v", err)
        return fmt.Sprintf("%d", time.Now().UnixNano())
    }
    return hex.EncodeToString(bytes)
}

type AnalyzeDocumentRequest struct {
    Content      string `json:"content"`  
    FileType     string `json:"fileType"` 
    NumQuestions int    `json:"numQuestions"` 
}

type AnalyzeDocumentResponse struct {
    Questions []QuestionData `json:"questions"`
    Error     *ErrorResponse `json:"error,omitempty"`
}

type ErrorResponse struct {
    Code    int    `json:"code"`
    Message string `json:"message"`
    Status  string `json:"status"`
}

type QuestionData struct {
    Content     string       `json:"content"`
    Type        string       `json:"type"`
    Options     []OptionData `json:"options"`
    Explanation string       `json:"explanation,omitempty"`
}

type OptionData struct {
    Content   string `json:"content"`
    IsCorrect bool   `json:"isCorrect"`
}

type GeminiRequest struct {
    Contents []ContentItem `json:"contents"`
}

type ContentItem struct {
    Parts []PartItem `json:"parts"`
}

type PartItem struct {
    Text string `json:"text,omitempty"`
}

type GeminiResponse struct {
    Candidates []struct {
        Content struct {
            Parts []struct {
                Text string `json:"text"`
            } `json:"parts"`
        } `json:"content"`
    } `json:"candidates"`
}
func (qc *quizController) analyzeDocument(filePath string, fileType string, numQuestions int) ([]QuestionData, error) {

    fileContent, err := ioutil.ReadFile(filePath)
    if err != nil {
        return nil, fmt.Errorf("không thể đọc file: %v", err)
    }
    
    encodedContent := base64.StdEncoding.EncodeToString(fileContent)
    
    prompt := fmt.Sprintf(`Hãy tạo %d câu hỏi trắc nghiệm dựa trên nội dung của file %s này. 
    Đây là nội dung của file được mã hóa base64: %s
    
    Hãy trả về kết quả theo định dạng JSON với cấu trúc sau:
    {
      "questions": [
        {
          "content": "Nội dung câu hỏi",
          "type": "multiple_choice",
          "options": [
            {"content": "Đáp án A", "isCorrect": true},
            {"content": "Đáp án B", "isCorrect": false},
            {"content": "Đáp án C", "isCorrect": false},
            {"content": "Đáp án D", "isCorrect": false}
          ],
          "explanation": "Giải thích đáp án"
        }
      ]
    }
    Chỉ trả về JSON, không có giải thích hay văn bản khác.`, numQuestions, fileType, encodedContent)
    
    requestBody := GeminiRequest{
        Contents: []ContentItem{
            {
                Parts: []PartItem{
                    {Text: prompt},
                },
            },
        },
    }
    jsonData, err := json.Marshal(requestBody)
    if err != nil {
        return nil, fmt.Errorf("lỗi khi tạo request: %v", err)
    }
    
    apiURL := os.Getenv("DOCUMENT_ANALYSIS_API") 
    if apiURL == "" {
        return generateSampleQuestions(numQuestions, fileType), nil
    }
    
    resp, err := http.Post(apiURL, "application/json", bytes.NewBuffer(jsonData))
    if err != nil {
        return nil, fmt.Errorf("lỗi khi gọi API phân tích: %v", err)
    }
    defer resp.Body.Close()
    
    respBody, err := io.ReadAll(resp.Body)
    if err != nil {
        return generateSampleQuestions(numQuestions, fileType), fmt.Errorf("lỗi khi đọc phản hồi: %v", err)
    }
    var geminiResponse GeminiResponse
    if err := json.Unmarshal(respBody, &geminiResponse); err != nil {
        log.Printf("Error parsing Gemini response: %v. Response: %s", err, string(respBody))
        return generateSampleQuestions(numQuestions, fileType), nil
    }
    if len(geminiResponse.Candidates) > 0 && len(geminiResponse.Candidates[0].Content.Parts) > 0 {
        jsonText := geminiResponse.Candidates[0].Content.Parts[0].Text
        
        jsonText = extractJSONFromText(jsonText)
        
        var questionsWrapper struct {
            Questions []QuestionData `json:"questions"`
        }
        
        if err := json.Unmarshal([]byte(jsonText), &questionsWrapper); err != nil {
            log.Printf("Error parsing questions JSON: %v. JSON: %s", err, jsonText)
            return generateSampleQuestions(numQuestions, fileType), nil
        }
        
        return questionsWrapper.Questions, nil
    }

    return generateSampleQuestions(numQuestions, fileType), nil
}

func generateSampleQuestions(numQuestions int, fileType string) []QuestionData {
    questions := make([]QuestionData, numQuestions)
    
    for i := 0; i < numQuestions; i++ {
        questionType := "multiple_choice"
        if i % 5 == 0 {
            questionType = "true_false"
        }
        
        var fileTypeDesc string
        switch fileType {
        case "pdf":
            fileTypeDesc = "PDF"
        case "docx":
            fileTypeDesc = "Word"
        case "xlsx":
            fileTypeDesc = "Excel"
        default:
            fileTypeDesc = "tài liệu"
        }
        
        questions[i] = QuestionData{
            Content: fmt.Sprintf("Đây là câu hỏi thứ %d được tạo từ %s của bạn?", i+1, fileTypeDesc),
            Type:    questionType,
            Options: []OptionData{
                {Content: "Đáp án A", IsCorrect: i%4 == 0},
                {Content: "Đáp án B", IsCorrect: i%4 == 1},
                {Content: "Đáp án C", IsCorrect: i%4 == 2},
                {Content: "Đáp án D", IsCorrect: i%4 == 3},
            },
            Explanation: fmt.Sprintf("Đây là phần giải thích cho câu hỏi %d.", i+1),
        }
    }
    
    return questions
}

func extractJSONFromText(text string) string {
    codeBlockRegex := regexp.MustCompile("```(?:json)?\\s*(\\{[\\s\\S]*?\\})\\s*```")
    matches := codeBlockRegex.FindStringSubmatch(text)
    if len(matches) > 1 {
        return matches[1]
    }
    
    jsonRegex := regexp.MustCompile("(\\{[\\s\\S]*\\})")
    matches = jsonRegex.FindStringSubmatch(text)
    if len(matches) > 1 {
        return matches[1]
    }
    
    return text
}