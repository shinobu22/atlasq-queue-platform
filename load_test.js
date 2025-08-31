import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate } from 'k6/metrics';

// Custom metrics
export const errorRate = new Rate('errors');

export const options = {
  stages: [
    { duration: '30s', target: 20 },   // Ramp up to 20 users over 30 seconds
    { duration: '1m', target: 50 },    // Stay at 50 users for 1 minute
    { duration: '1m', target: 100 },   // Ramp up to 100 users for 1 minute
    { duration: '2m', target: 100 },   // Stay at 100 users for 2 minutes
    { duration: '30s', target: 0 },    // Ramp down to 0 users
  ],
  thresholds: {
    http_req_duration: ['p(95)<500'], // 95% of requests should be under 500ms
    http_req_failed: ['rate<0.1'],    // Error rate should be less than 10%
    errors: ['rate<0.1'],             // Custom error rate should be less than 10%
  },
};

const BASE_URL = 'http://localhost:8083';

// Generate random order data
function generateOrderData() {
  const orderNumber = `ORD-${Date.now()}-${Math.floor(Math.random() * 10000)}`;
  const quantity = Math.floor(Math.random() * 100) + 1; // 1-100 items
  
  return {
    order_number: orderNumber,
    quantity: quantity,
  };
}

export default function () {
  // Generate random order data for this iteration
  const orderData = generateOrderData();
  
  const payload = JSON.stringify(orderData);
  const params = {
    headers: {
      'Content-Type': 'application/json',
    },
  };

  // Make POST request to /v1/orders endpoint
  const response = http.post(`${BASE_URL}/v1/orders`, payload, params);
  
  // Check response
  const result = check(response, {
    'status is 200': (r) => r.status === 200,
    'response time < 500ms': (r) => r.timings.duration < 500,
    'response body contains success': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.status === 'success';
      } catch (e) {
        return false;
      }
    },
    'content-type is application/json': (r) => r.headers['Content-Type'] === 'application/json',
  });

  // Record errors
  errorRate.add(!result);

  // Log details for failed requests
  if (response.status !== 200) {
    console.error(`Request failed: ${response.status} ${response.statusText}`);
    console.error(`Response: ${response.body}`);
    console.error(`Order: ${orderData.order_number}, Quantity: ${orderData.quantity}`);
  }

  // Brief pause between requests
  sleep(Math.random() * 2 + 0.5); // Random sleep between 0.5-2.5 seconds
}

// Setup function to verify the endpoint is available before starting the test
export function setup() {
  console.log('Setting up load test...');
  
  // Test endpoint availability
  const response = http.get(`${BASE_URL}/healthz`);
  
  if (response.status !== 200) {
    throw new Error(`Setup failed: API is not available (status: ${response.status})`);
  }
  
  console.log('API is available, starting load test...');
  
  // Return data that will be passed to the default function
  return {
    startTime: new Date().toISOString(),
  };
}

// Teardown function to log summary
export function teardown(data) {
  console.log(`Load test completed. Started at: ${data.startTime}`);
  console.log('Check the k6 metrics above for detailed results.');
}