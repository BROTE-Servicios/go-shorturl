package main

import (
    "crypto/sha256"
    "database/sql"
    "encoding/base64"
    "log"
    "net/http"
    "os"
    "strings"

    "github.com/gin-gonic/gin"
    _ "github.com/mattn/go-sqlite3"
)

const base62 = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

type AcortarRequest struct {
    URL string `json:"url"`
}

func generarClaveCorta(url string) string {
    hasher := sha256.New()
    hasher.Write([]byte(url))
    hash := hasher.Sum(nil)
    base64Encoded := base64.StdEncoding.EncodeToString(hash)[:10]
    return base64ToBase62(base64Encoded)
}

func base64ToBase62(s string) string {
    var result string
    for _, c := range s {
        if c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9' {
            result += string(c)
        } else {
            result += string(base62[int(c)%len(base62)])
        }
    }
    return result
}

func initDB() *sql.DB {
    db, err := sql.Open("sqlite3", "./urls.db")
    if err != nil {
        log.Fatal(err)
    }

    statement, _ := db.Prepare(`CREATE TABLE IF NOT EXISTS urls (
        id INTEGER PRIMARY KEY,
        urlOriginal TEXT,
        urlAcortada TEXT
    )`)
    statement.Exec()
    return db
}

func insertURL(db *sql.DB, urlOriginal, urlAcortada string) error {
    statement, err := db.Prepare("INSERT INTO urls (urlOriginal, urlAcortada) VALUES (?, ?)")
    if err != nil {
        return err
    }
    _, err = statement.Exec(urlOriginal, urlAcortada)
    return err
}

func getOriginalURL(db *sql.DB, urlAcortada string) (string, error) {
    var urlOriginal string
    row := db.QueryRow("SELECT urlOriginal FROM urls WHERE urlAcortada = ?", urlAcortada)
    err := row.Scan(&urlOriginal)
    return urlOriginal, err
}

func main() {
    puerto := os.Getenv("PORT")
    if puerto == "" {
        puerto = "8085"
    }

    db := initDB()
    defer db.Close()

    router := gin.Default()

    // Sirve archivos estáticos desde el directorio 'public'
    router.Static("/static", "./public")

    // Sirve el index.html como página de inicio
    router.GET("/", func(c *gin.Context) {
        c.File("./public/index.html")
    })

    router.POST("/acortar", func(c *gin.Context) {
        var req AcortarRequest
        if err := c.BindJSON(&req); err != nil {
            c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
            return
        }

        claveCorta := generarClaveCorta(req.URL)
        err := insertURL(db, req.URL, claveCorta)
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al guardar en la base de datos"})
            return
        }

        c.JSON(http.StatusOK, gin.H{"urlAcortada": claveCorta})
    })

    router.GET("/:claveCorta", func(c *gin.Context) {
        claveCorta := c.Param("claveCorta")
        urlOriginal, err := getOriginalURL(db, claveCorta)
        if err != nil {
            c.JSON(http.StatusNotFound, gin.H{"error": "URL no encontrada"})
            return
        }

        if !strings.Contains(urlOriginal, "brote") {
            c.JSON(http.StatusNotFound, gin.H{"mensaje": "Servicio en desarrollo. Contacto: desarrollo@brote.org"})
            return
        }

        c.Redirect(http.StatusMovedPermanently, urlOriginal)
    })

    router.Run(":" + puerto)
}
