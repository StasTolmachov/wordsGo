import http from 'k6/http';
import { sleep } from 'k6';

export const options = {
    stages: [
        { duration: '1m', target: 100 },  // Разгон: до 50 пользователей за 1 минуту
        { duration: '3m', target: 100 },  // Плато: держим 50 пользователей 3 минуты
        { duration: '1m', target: 0 },   // Снижение: до 0 пользователей
    ],
};

export default function () {
    // Замените на ваш URL в DigitalOcean
    const url = 'https://wordsgo.tolmachov.dev/api/v1/wordsGo/words';

    // Если эндпоинт защищен JWT
    const params = {
        headers: {
            'Authorization': 'Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoiYWZlOGNiN2EtOGE3OS00YmQ1LTkwYmItMTUzZmQ5MzA5NzBlIiwicm9sZSI6IiIsImV4cCI6MTc2OTU4NzU0NSwiaWF0IjoxNzY5NTAxMTQ1fQ.I5vnyY8kf4NJOgbaAA6GN70Mb72D-d-wj10JCrPNGR0',
        },
    };

    http.get(url, params);
    sleep(1); // Пауза между запросами имитирует чтение пользователем
}