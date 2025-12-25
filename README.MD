# Document-flow v2fdsf


## Пересобрать postgre_task-manager
```bash
dcd -v postgres-task-manager #удалить вместо с томом даннх. НА ПРОДЕ НЕ УДАЛЯТЬ ТОМ ДЕЛАТЬ!
dcu --build postgres-task-manager
make migrate-up
```