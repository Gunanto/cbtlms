import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  scenarios: {
    autosave_submit: {
      executor: 'ramping-vus',
      startVUs: 1,
      stages: [
        { duration: '30s', target: 20 },
        { duration: '60s', target: 50 },
        { duration: '30s', target: 0 },
      ],
      gracefulRampDown: '10s',
    },
  },
  thresholds: {
    http_req_failed: ['rate<0.05'],
    http_req_duration: ['p(95)<300'],
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const ATTEMPT_ID = __ENV.ATTEMPT_ID || '1';
const QUESTION_ID = __ENV.QUESTION_ID || '1';
const COOKIE = __ENV.COOKIE || '';
const CSRF_TOKEN = __ENV.CSRF_TOKEN || '';

function authHeaders() {
  const headers = {
    'Content-Type': 'application/json',
  };
  if (COOKIE) {
    headers['Cookie'] = COOKIE;
  }
  if (CSRF_TOKEN) {
    headers['X-CSRF-Token'] = CSRF_TOKEN;
  }
  return headers;
}

export default function () {
  const headers = authHeaders();

  const saveRes = http.put(
    `${BASE_URL}/api/v1/attempts/${ATTEMPT_ID}/answers/${QUESTION_ID}`,
    JSON.stringify({
      answer_payload: { selected: ['B'] },
      is_doubt: false,
    }),
    { headers }
  );

  check(saveRes, {
    'autosave status is 200': (r) => r.status === 200,
  });

  const summaryRes = http.get(`${BASE_URL}/api/v1/attempts/${ATTEMPT_ID}`, { headers });
  check(summaryRes, {
    'summary status is 200': (r) => r.status === 200,
  });

  if (__ITER % 25 === 0) {
    const submitRes = http.post(`${BASE_URL}/api/v1/attempts/${ATTEMPT_ID}/submit`, null, { headers });
    check(submitRes, {
      'submit status is 200': (r) => r.status === 200,
    });
  }

  sleep(0.2);
}
