package migrations

import (
	"log"

	_ "github.com/lib/pq"

	"github.com/pressly/goose"
)

// RunMigrations выполняет миграции базы данных
func RunMigrations(dbInfo string) {
	db, err := goose.OpenDBWithDriver("postgres", dbInfo)
	if err != nil {
		log.Fatalf("Не удалось подключиться к базе данных: %v", err)
	}
	defer db.Close() // Закрываем соединение после выполнения миграций

	log.Println("Запуск миграций...")
	if err := goose.Up(db, "../../migrations"); err != nil {
		log.Fatalf("Ошибка выполнения миграций: %v", err)
	}

	log.Println("Миграции выполнены успешно.")
}
