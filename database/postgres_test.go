package database

import (
    "context"
//    "database/sql"
    "testing"

    "github.com/DATA-DOG/go-sqlmock"
    "github.com/edamsoft-sre/alpaca/models"
    "github.com/stretchr/testify/assert"
)

// No value in adding injection to properly test this for now.
// func TestNewPostgresRepository(t *testing.T) {
    // db, _, err := sqlmock.New()
    // if err != nil {
    //     t.Fatalf("Error al crear sqlmock: %s", err)
    // }
    // defer db.Close()

    // repo := &PostgresRepository{db: db}
    // assert.NotNil(t, repo)
//}

func TestInsertUser(t *testing.T) {
    db, mock, err := sqlmock.New()
    if err != nil {
        t.Fatalf("Error al crear sqlmock: %s", err)
    }
    defer db.Close()

    repo := &PostgresRepository{db: db}
    user := &models.User{
        Id:       "123",
        Email:    "test@example.com",
        Password: "password123",
    }

    mock.ExpectExec("INSERT INTO users").
        WithArgs(user.Email, user.Password, user.Id).
        WillReturnResult(sqlmock.NewResult(1, 1))

    err = repo.InsertUser(context.Background(), user)
    assert.NoError(t, err)
    assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetUserByID(t *testing.T) {
    db, mock, err := sqlmock.New()
    if err != nil {
        t.Fatalf("Error al crear sqlmock: %s", err)
    }
    defer db.Close()

    repo := &PostgresRepository{db: db}
    userID := "123"
    expectedUser := &models.User{
        Id:    "123",
        Email: "test@example.com",
    }

    rows := sqlmock.NewRows([]string{"id", "email"}).
        AddRow(expectedUser.Id, expectedUser.Email)

    mock.ExpectQuery("SELECT id, email FROM users WHERE id = \\$1").
        WithArgs(userID).
        WillReturnRows(rows)

    user, err := repo.GetUserByID(context.Background(), userID)
    assert.NoError(t, err)
    assert.Equal(t, expectedUser, user)
    assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDeletePost(t *testing.T) {
    db, mock, err := sqlmock.New()
    if err != nil {
        t.Fatalf("Error al crear sqlmock: %s", err)
    }
    defer db.Close()

    repo := &PostgresRepository{db: db}
    postID := "456"
    userID := "123"

    mock.ExpectExec("DELETE FROM posts WHERE id = \\$1 and user_id = \\$2").
        WithArgs(postID, userID).
        WillReturnResult(sqlmock.NewResult(0, 1))

    err = repo.DeletePost(context.Background(), postID, userID)
    assert.NoError(t, err)
    assert.NoError(t, mock.ExpectationsWereMet())
}