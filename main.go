package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
)

// Todo represents a single todo item
type Todo struct {
	ID    int    `json:"id"`
	Title string `json:"title"`
	Done  bool   `json:"done"`
}

var (
	todos  []Todo
	nextID int
	mu     sync.Mutex
)

func init() {
	todos = []Todo{
		{ID: 1, Title: "Learn Go", Done: false},
		{ID: 2, Title: "Learn Jujutsu", Done: false},
		{ID: 3, Title: "Build a simple API", Done: true},
	}
	nextID = 4
}

func useTemplate(name, tmpl string, w http.ResponseWriter, data any) {
	t, err := template.New(name).Parse(tmpl)
	if err != nil {
		http.Error(w, "Failed to parse template", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	if err := t.Execute(w, data); err != nil {
		http.Error(w, "Failed to execute template", http.StatusInternalServerError)
	}
}

func todoHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPut:
		updateTodo(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func updateTodo(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/todo/")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid todo ID", http.StatusBadRequest)
		return
	}

	var updatedTodo Todo
	if err := json.NewDecoder(r.Body).Decode(&updatedTodo); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	mu.Lock()
	defer mu.Unlock()

	for i, todo := range todos {
		if todo.ID == id {
			todos[i].Title = updatedTodo.Title
			todos[i].Done = updatedTodo.Done
			w.WriteHeader(http.StatusOK)
			return
		}
	}

	http.Error(w, "Todo not found", http.StatusNotFound)
}

func todosHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		getTodos(w, r)
	case http.MethodPost:
		createTodo(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func getTodos(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	defer mu.Unlock()

	tmpl := `<!DOCTYPE html>
<html>
<head>
	<title>Todos</title>
	<style>
		.view-mode { display: block; }
		.edit-mode { display: none; }
	</style>
</head>
<body>
	<h1>Todos</h1>
	<ul>
		{{range .}}
			<li id="todo-{{.ID}}">
				<form onsubmit="submitTodo({{.ID}}); return false;">
					<div class="view-mode">
						<input type="checkbox" {{if .Done}}checked{{end}} disabled>
						<span>{{.Title}}</span>
						<button type="button" onclick="toggleEdit({{.ID}})">Edit</button>
					</div>
					<div class="edit-mode">
						<input type="checkbox" id="done-{{.ID}}" {{if .Done}}checked{{end}}>
						<input type="text" id="title-{{.ID}}" value="{{.Title}}" required>
						<button type="submit">Submit</button>
					</div>
				</form>
			</li>
		{{end}}
	</ul>

	<script>
		function toggleEdit(id) {
			const todoLi = document.getElementById('todo-' + id);
			const viewMode = todoLi.querySelector('.view-mode');
			const editMode = todoLi.querySelector('.edit-mode');

			viewMode.style.display = 'none';
			editMode.style.display = 'block';
		}

		function submitTodo(id) {
			const titleInput = document.getElementById('title-' + id);
			const doneCheckbox = document.getElementById('done-' + id);

			const data = {
				title: titleInput.value,
				done: doneCheckbox.checked
			};

			fetch('/todo/' + id, {
				method: 'PUT',
				headers: {
					'Content-Type': 'application/json'
				},
				body: JSON.stringify(data)
			})
			.then(response => {
				if (response.ok) {
					// Switch back to view mode and update the content
					const todoLi = document.getElementById('todo-' + id);
					const viewMode = todoLi.querySelector('.view-mode');
					const editMode = todoLi.querySelector('.edit-mode');
					
					viewMode.querySelector('span').textContent = data.title;
					viewMode.querySelector('input[type="checkbox"]').checked = data.done;

					editMode.style.display = 'none';
					viewMode.style.display = 'block';
				} else {
					alert('Failed to update todo');
				}
			})
			.catch(error => {
				console.error('Error:', error);
				alert('An error occurred');
			});
		}
	</script>
</body>
</html>`

	useTemplate("todos", tmpl, w, todos)
}

func createTodo(w http.ResponseWriter, r *http.Request) {
	var todo Todo
	if err := json.NewDecoder(r.Body).Decode(&todo); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	mu.Lock()
	defer mu.Unlock()

	todo.ID = nextID
	nextID++
	todos = append(todos, todo)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(todo); err != nil {
		http.Error(w, "Failed to encode created todo", http.StatusInternalServerError)
	}
}

func main() {
	http.HandleFunc("/todos", todosHandler)
	http.HandleFunc("/todo/", todoHandler)

	fmt.Println("Server starting on http://localhost:8080...")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Could not start server: %s\n", err)
	}
}
