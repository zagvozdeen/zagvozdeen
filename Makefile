.PHONY: deploy

deploy:
	npm run build && go run cmd/converter/main.go
	GOOS=linux GOARCH=amd64 go build -o zagvozdeen ./cmd/api
	ssh root@185.221.214.4 "systemctl stop zagvozdeen.service && rm -rf /var/www/zagvozdeen.ru && mkdir /var/www/zagvozdeen.ru"
	scp zagvozdeen root@185.221.214.4:/var/www/zagvozdeen.ru
	scp -r dist root@185.221.214.4:/var/www/zagvozdeen.ru/dist
	scp -r public root@185.221.214.4:/var/www/zagvozdeen.ru/public
	scp deploy/nginx.conf root@185.221.214.4:/etc/nginx/sites-available/zagvozdeen.ru
#	ssh root@185.221.214.4 "ln -s /etc/nginx/sites-available/zagvozdeen.ru /etc/nginx/sites-enabled/zagvozdeen.ru"
	scp deploy/zagvozdeen.service root@185.221.214.4:/etc/systemd/system
	ssh root@185.221.214.4 "systemctl daemon-reload && systemctl restart zagvozdeen.service"
