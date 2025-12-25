## Как скачать docker образ


1) Порт ssh изменен на кастомный, для постноянной кофигурации лучше указать в .ssh конфигуарцию: 
```ini
Host gitea
    HostName localhost
    Port 222 # порт ssh на сервере
    User git # user для авторизации
    IdentityFile ~/.ssh/id_ed25519 # ssh ключ
```
 
2) В частный регистр нужно логинится: 
```bash
docker login gitea.example.com 
#пример 
docker login 192.168.78.61:4925 #ввести логин/пароль от аккаунта gitea
```
    ввести логин пароль на gitea
3) При загрузке образов нужно указывать перед именем пользователя, домен регестри, с выгрузкой аналогично:
```bash
docker build -t {registry}/{owner}/{image}:{tag} .  
# name an existing image with tag  
docker tag {some-existing-image}:{tag} {registry}/{owner}/{image}:{tag}
## gitea.example.com/testuser/myimage

docker push gitea.example.com/{owner}/{image}:{tag}


#пример
docker build -t document-flow-task-manager:latest .
docker tag document-flow-task-manager:latest 192.168.78.61:4925/ravil-developer/task-manager:v1
docker push 192.168.78.61:4925/ravil-developer/task-manager:v1