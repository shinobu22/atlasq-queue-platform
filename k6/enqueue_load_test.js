import http from 'k6/http';
import { check, sleep } from 'k6';
import { Counter, Rate, Trend } from 'k6/metrics';

// Custom metrics
const enqueuedTasks = new Counter('enqueued_tasks_total');
const enqueueErrors = new Counter('enqueue_errors_total');
const enqueueSuccessRate = new Rate('enqueue_success_rate');
const enqueueDuration = new Trend('enqueue_duration');

// Test configuration
export const options = {
  stages: [
    { duration: '30s', target: 5 },   // Ramp up to 5 users
    { duration: '1m', target: 10 },   // Stay at 10 users
    { duration: '30s', target: 20 },  // Ramp up to 20 users
    { duration: '2m', target: 20 },   // Stay at 20 users
    { duration: '30s', target: 0 },   // Ramp down
  ],
  thresholds: {
    http_req_duration: ['p(95)<2000'], // 95% of requests must complete below 2s
    enqueue_success_rate: ['rate>0.95'], // 95% success rate
    http_req_failed: ['rate<0.05'], // Less than 5% error rate
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

// Sample payloads
const emailPayloads = [
  {
    to: 'user1@example.com',
    subject: 'Load Test Email 1',
    body: 'This is a test email from k6 load test.'
  },
  {
    to: 'user2@test.org',
    subject: 'Performance Test Notification',
    body: 'Testing the email system under load. Message contains important performance metrics.'
  },
  {
    to: 'admin@demo.net',
    subject: 'System Health Check',
    body: 'Automated system health check email generated during load testing.'
  }
];

const reportPayloads = [
  {
    report_id: `load-test-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`,
    params: {
      type: 'performance',
      period: 'daily',
      format: 'json',
      created_by: 'k6-load-test'
    }
  },
  {
    report_id: `analytics-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`,
    params: {
      type: 'analytics',
      period: 'weekly',
      format: 'pdf',
      filters: {
        date_range: {
          start: '2024-01-01',
          end: '2024-01-31'
        }
      }
    }
  }
];

const queues = ['critical', 'default', 'low'];
const delays = ['5s', '10s', '30s', '1m'];

export default function() {
  // Randomly choose task type (70% email, 30% report)
  const isEmailTask = Math.random() < 0.7;
  
  let taskType, payload;
  if (isEmailTask) {
    taskType = 'send_email';
    payload = emailPayloads[Math.floor(Math.random() * emailPayloads.length)];
  } else {
    taskType = 'generate_report';
    payload = reportPayloads[Math.floor(Math.random() * reportPayloads.length)];
  }

  // Build URL with random parameters
  let url = `${BASE_URL}/enqueue/${taskType}`;
  const params = new URLSearchParams();
  
  // 30% chance of adding queue parameter
  if (Math.random() < 0.3) {
    params.append('queue', queues[Math.floor(Math.random() * queues.length)]);
  }
  
  // 20% chance of adding delay
  if (Math.random() < 0.2) {
    params.append('delay', delays[Math.floor(Math.random() * delays.length)]);
  }
  
  // 10% chance of adding max_retry
  if (Math.random() < 0.1) {
    params.append('max_retry', String(Math.floor(Math.random() * 5) + 3));
  }
  
  if (params.toString()) {
    url += '?' + params.toString();
  }

  // Prepare request
  const headers = {
    'Content-Type': 'application/json',
  };
  
  // 20% chance of adding idempotency key
  if (Math.random() < 0.2) {
    headers['X-Idempotency-Key'] = `k6-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;
  }

  const requestPayload = JSON.stringify({ payload: payload });

  // Make request
  const response = http.post(url, requestPayload, { headers: headers });
  
  // Record metrics
  enqueueDuration.add(response.timings.duration);
  
  // Check response
  const success = check(response, {
    'status is 200': (r) => r.status === 200,
    'response has task_id': (r) => {
      const body = JSON.parse(r.body);
      return body.task_id && body.task_id.length > 0;
    },
    'response has queue': (r) => {
      const body = JSON.parse(r.body);
      return body.queue && ['critical', 'default', 'low'].includes(body.queue);
    },
    'enqueued is true (unless existing)': (r) => {
      const body = JSON.parse(r.body);
      return body.enqueued === true || body.existing === true;
    },
  });
  
  if (success) {
    enqueuedTasks.add(1);
    enqueueSuccessRate.add(1);
  } else {
    enqueueErrors.add(1);
    enqueueSuccessRate.add(0);
    console.log(`Request failed: ${response.status} ${response.body}`);
  }

  // Random sleep between requests (0.1s to 1s)
  sleep(Math.random() * 0.9 + 0.1);
}

export function handleSummary(data) {
  return {
    'stdout': textSummary(data, { indent: ' ', enableColors: true }),
    'k6-results.json': JSON.stringify(data),
  };
}

function textSummary(data, options = {}) {
  const indent = options.indent || '';
  const colors = options.enableColors !== false;
  
  let summary = `
${indent}Load Test Summary
${indent}================
${indent}
${indent}Scenarios:
${indent}  Default: ${data.metrics.vus?.values?.value || 0} VUs for ${data.state?.testRunDurationMs / 1000 || 0}s
${indent}
${indent}Key Metrics:
${indent}  HTTP Requests: ${data.metrics.http_reqs?.values?.count || 0}
${indent}  HTTP Request Duration (p95): ${data.metrics.http_req_duration?.values?.['p(95)']?.toFixed(2) || 0}ms
${indent}  HTTP Request Failed: ${((data.metrics.http_req_failed?.values?.rate || 0) * 100).toFixed(2)}%
${indent}  
${indent}Custom Metrics:
${indent}  Tasks Enqueued: ${data.metrics.enqueued_tasks_total?.values?.count || 0}
${indent}  Enqueue Errors: ${data.metrics.enqueue_errors_total?.values?.count || 0}
${indent}  Success Rate: ${((data.metrics.enqueue_success_rate?.values?.rate || 0) * 100).toFixed(2)}%
${indent}  Enqueue Duration (avg): ${data.metrics.enqueue_duration?.values?.avg?.toFixed(2) || 0}ms
${indent}
`;

  return summary;
}