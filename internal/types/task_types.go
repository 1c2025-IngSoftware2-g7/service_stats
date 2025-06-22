package types

/*
	How to add a new task type:

1. Add a new constant for the task type, you can call it as you like
2. Go to queue/handler.go, and add a new handler function for the task type
3. Register the new handler function in the NewMux function
4. Add into the API requeset to the queue the new task type

And you're done! Regards lucas!
*/
const TaskAddStudentGrade = "task:add_student_grade"
