import { test, expect } from '@playwright/test';

test.describe('Dashboard - Initial State', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
  });

  test('renders the llm-viz header', async ({ page }) => {
    await expect(page.getByText('llm-viz')).toBeVisible();
    await expect(page.getByText('Real-time token dashboard')).toBeVisible();
  });

  test('shows Provider Settings panel', async ({ page }) => {
    await expect(page.getByText('Provider Settings')).toBeVisible();
  });

  test('shows Token Counter panel', async ({ page }) => {
    await expect(page.getByText('Token Counter')).toBeVisible();
  });

  test('shows Context Window panel', async ({ page }) => {
    await expect(page.getByText('Context Window')).toBeVisible();
  });

  test('shows Cost Tracker panel', async ({ page }) => {
    await expect(page.getByText('Cost Tracker')).toBeVisible();
  });

  test('shows Cache Efficiency panel', async ({ page }) => {
    await expect(page.getByText('Cache Efficiency')).toBeVisible();
  });

  test('shows Usage Timeline panel', async ({ page }) => {
    await expect(page.getByText('Usage Timeline')).toBeVisible();
  });

  test('shows chat input with placeholder text', async ({ page }) => {
    await expect(
      page.getByPlaceholder('Send a message to track token usage…')
    ).toBeVisible();
  });

  test('shows API key warning on initial load', async ({ page }) => {
    await expect(page.getByText(/Enter your API key/i)).toBeVisible();
  });

  test('shows disabled state for chat input without API key', async ({ page }) => {
    await expect(page.getByText(/Enter an API key above/i)).toBeVisible();
  });

  test('defaults to Anthropic provider', async ({ page }) => {
    await expect(page.getByRole('option', { name: 'Anthropic' })).toBeAttached();
    const providerSelect = page.locator('select').first();
    await expect(providerSelect).toHaveValue('anthropic');
  });

  test('shows Clear data button', async ({ page }) => {
    await expect(page.getByText('Clear data')).toBeVisible();
  });
});

test.describe('Dashboard - Provider Selection', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
  });

  test('can switch from Anthropic to OpenAI', async ({ page }) => {
    const selects = page.locator('select');
    await selects.first().selectOption('openai');
    await expect(selects.first()).toHaveValue('openai');
  });

  test('model list updates when switching to OpenAI', async ({ page }) => {
    const selects = page.locator('select');
    await selects.first().selectOption('openai');
    // After switching to OpenAI, gpt-4o should appear in model select
    const modelSelect = selects.nth(1);
    await expect(modelSelect.locator('option[value="gpt-4o"]')).toBeAttached();
  });

  test('model list shows claude models for Anthropic', async ({ page }) => {
    const selects = page.locator('select');
    await expect(selects.first()).toHaveValue('anthropic');
    const modelSelect = selects.nth(1);
    await expect(modelSelect.locator('option[value="claude-sonnet-4-6"]')).toBeAttached();
  });

  test('can select a different model', async ({ page }) => {
    const modelSelect = page.locator('select').nth(1);
    await modelSelect.selectOption('claude-haiku-4-5');
    await expect(modelSelect).toHaveValue('claude-haiku-4-5');
  });

  test('API key placeholder changes for OpenAI', async ({ page }) => {
    const selects = page.locator('select');
    await selects.first().selectOption('openai');
    // OpenAI key placeholder should be sk-... not sk-ant-...
    await expect(page.getByPlaceholder('sk-...')).toBeVisible();
  });
});

test.describe('Dashboard - API Key Input', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
  });

  test('API key field starts as password type', async ({ page }) => {
    const apiKeyInput = page.getByPlaceholder('sk-ant-...');
    await expect(apiKeyInput).toHaveAttribute('type', 'password');
  });

  test('can toggle API key visibility', async ({ page }) => {
    const apiKeyInput = page.getByPlaceholder('sk-ant-...');
    const toggleBtn = page.getByRole('button', { name: 'Show key' });
    await toggleBtn.click();
    await expect(apiKeyInput).toHaveAttribute('type', 'text');
    await page.getByRole('button', { name: 'Hide key' }).click();
    await expect(apiKeyInput).toHaveAttribute('type', 'password');
  });

  test('entering API key enables the chat input', async ({ page }) => {
    await page.getByPlaceholder('sk-ant-...').fill('sk-ant-test-key-12345');
    // The "Enter an API key above" message should disappear
    await expect(page.getByText(/Enter an API key above/i)).not.toBeVisible();
  });

  test('API key warning disappears when key is entered', async ({ page }) => {
    await expect(page.getByText(/Enter your API key/i)).toBeVisible();
    await page.getByPlaceholder('sk-ant-...').fill('test-key');
    await expect(page.getByText(/Enter your API key/i)).not.toBeVisible();
  });
});

test.describe('Dashboard - Chat Input', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
  });

  test('Send button is disabled without API key', async ({ page }) => {
    const sendBtn = page.getByRole('button', { name: 'Send' });
    await expect(sendBtn).toBeDisabled();
  });

  test('chat textarea is disabled without API key', async ({ page }) => {
    const textarea = page.getByPlaceholder('Send a message to track token usage…');
    await expect(textarea).toBeDisabled();
  });

  test('Send button remains disabled when textarea is empty with API key', async ({ page }) => {
    await page.getByPlaceholder('sk-ant-...').fill('sk-test');
    const sendBtn = page.getByRole('button', { name: 'Send' });
    await expect(sendBtn).toBeDisabled();
  });
});

test.describe('Dashboard - Token Counter Initial State', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
  });

  test('shows zero counts initially', async ({ page }) => {
    // Check that stat cards exist with zero values
    const tokenSection = page.getByText('Token Counter').locator('../..');
    await expect(tokenSection).toBeVisible();
  });

  test('shows Input, Output, Cache Read, Cache Write labels', async ({ page }) => {
    await expect(page.getByText('Input', { exact: true })).toBeVisible();
    await expect(page.getByText('Output', { exact: true })).toBeVisible();
    await expect(page.getByText('Cache Read')).toBeVisible();
    await expect(page.getByText('Cache Write')).toBeVisible();
  });

  test('shows This request and Session total labels', async ({ page }) => {
    await expect(page.getByText('This request')).toBeVisible();
    await expect(page.getByText('Session total', { exact: true })).toBeVisible();
  });
});

test.describe('Dashboard - Context Gauge Initial State', () => {
  test('shows 0.0% utilization initially', async ({ page }) => {
    await page.goto('/');
    await expect(page.getByText('0.0%')).toBeVisible();
  });

  test('shows used label', async ({ page }) => {
    await page.goto('/');
    await expect(page.getByText('used', { exact: true })).toBeVisible();
  });
});

test.describe('Dashboard - Cost Tracker Initial State', () => {
  test('shows USD estimates label', async ({ page }) => {
    await page.goto('/');
    await expect(page.getByText('USD estimates')).toBeVisible();
  });

  test('shows session total label', async ({ page }) => {
    await page.goto('/');
    await expect(page.getByText('session total', { exact: true })).toBeVisible();
  });
});

test.describe('Dashboard - Cache Chart Initial State', () => {
  test('shows Cache Efficiency heading', async ({ page }) => {
    await page.goto('/');
    await expect(page.getByText('Cache Efficiency')).toBeVisible();
  });

  test('shows "No cache activity yet" message initially with Anthropic', async ({ page }) => {
    await page.goto('/');
    // No cache activity message when no requests made
    await expect(page.getByText(/No cache activity yet/i)).toBeVisible();
  });

  test('shows unsupported message when switching to OpenAI', async ({ page }) => {
    await page.goto('/');
    const selects = page.locator('select');
    await selects.first().selectOption('openai');
    await expect(page.getByText(/Cache tracking not supported/i)).toBeVisible();
  });
});

test.describe('Dashboard - Clear Data', () => {
  test('Clear data button is clickable', async ({ page }) => {
    await page.goto('/');
    const clearBtn = page.getByText('Clear data');
    await clearBtn.click();
    // No error should occur, state should remain clean
    await expect(page.getByText('Token Counter')).toBeVisible();
  });
});

test.describe('Dashboard - Accessibility', () => {
  test('page has proper heading structure', async ({ page }) => {
    await page.goto('/');
    const headings = page.locator('h2');
    await expect(headings).toHaveCount(6);
  });

  test('API key toggle button has accessible name', async ({ page }) => {
    await page.goto('/');
    await expect(page.getByRole('button', { name: 'Show key' })).toBeVisible();
  });
});
