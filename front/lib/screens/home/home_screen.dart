import 'package:flutter/material.dart';
import '../../models/todo.dart';
import '../../services/auth_service.dart';
import '../../services/todo_service.dart';

class HomeScreen extends StatefulWidget {
  const HomeScreen({super.key});

  @override
  State<HomeScreen> createState() => _HomeScreenState();
}

class _HomeScreenState extends State<HomeScreen> {
  final _todoService = TodoService();
  final _authService = AuthService();
  final _newTodoController = TextEditingController();
  List<Todo> _todos = [];
  bool _isLoading = true;

  @override
  void initState() {
    super.initState();
    _loadTodos();
  }

  @override
  void dispose() {
    _newTodoController.dispose();
    super.dispose();
  }

  Future<void> _loadTodos() async {
    try {
      final todos = await _todoService.getTodos();
      setState(() {
        _todos = todos;
        _isLoading = false;
      });
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('Error loading todos: ${e.toString()}')),
        );
      }
    }
  }

  Future<void> _handleSignOut(BuildContext context) async {
    await _authService.removeToken();
    if (mounted) {
      Navigator.of(context).pushReplacementNamed('/');
    }
  }

  Future<void> _createTodo() async {
    if (_newTodoController.text.trim().isEmpty) return;

    try {
      final todo =
          await _todoService.createTodo(_newTodoController.text.trim());
      setState(() {
        _todos.add(todo);
      });
      _newTodoController.clear();
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('Error creating todo: ${e.toString()}')),
        );
      }
    }
  }

  Future<void> _toggleTodoStatus(Todo todo) async {
    try {
      final updatedTodo = await _todoService.toggleTodoStatus(
        todo.id,
        !todo.isCompleted,
      );
      setState(() {
        final index = _todos.indexWhere((t) => t.id == todo.id);
        if (index != -1) {
          _todos[index] = updatedTodo;
        }
      });
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('Error updating todo: ${e.toString()}')),
        );
      }
    }
  }

  Future<void> _deleteTodo(Todo todo) async {
    try {
      await _todoService.deleteTodo(todo.id);
      setState(() {
        _todos.removeWhere((t) => t.id == todo.id);
      });
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('Error deleting todo: ${e.toString()}')),
        );
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Bestodo'),
        actions: [
          IconButton(
            icon: const Icon(Icons.logout),
            onPressed: () => _handleSignOut(context),
          ),
        ],
      ),
      body: _isLoading
          ? const Center(child: CircularProgressIndicator())
          : Column(
              children: [
                Padding(
                  padding: const EdgeInsets.all(16.0),
                  child: Row(
                    children: [
                      Expanded(
                        child: TextField(
                          controller: _newTodoController,
                          decoration: const InputDecoration(
                            hintText: 'Add a new todo',
                            border: OutlineInputBorder(),
                          ),
                          onSubmitted: (_) => _createTodo(),
                        ),
                      ),
                      const SizedBox(width: 16),
                      ElevatedButton(
                        onPressed: _createTodo,
                        child: const Text('Add'),
                      ),
                    ],
                  ),
                ),
                Expanded(
                  child: ListView.builder(
                    itemCount: _todos.length,
                    itemBuilder: (context, index) {
                      final todo = _todos[index];
                      return ListTile(
                        leading: Checkbox(
                          value: todo.isCompleted,
                          onChanged: (_) => _toggleTodoStatus(todo),
                        ),
                        title: Text(
                          todo.title,
                          style: TextStyle(
                            decoration: todo.isCompleted
                                ? TextDecoration.lineThrough
                                : null,
                          ),
                        ),
                        trailing: IconButton(
                          icon: const Icon(Icons.delete),
                          onPressed: () => _deleteTodo(todo),
                        ),
                      );
                    },
                  ),
                ),
              ],
            ),
    );
  }
}
