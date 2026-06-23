export const API_URL = process.env.REACT_APP_API_URL || 'http://localhost:8080';

export interface TaskSubmitParams {
  title: string;
  context: string;
  difficulty: number;
  approvalRequired: boolean;
  referencedSkills?: string[];
}

export interface TaskResponse {
  taskID: string;
  status: string;
  error?: string;
}

export interface SkillsResponse {
  skills: any[];
  total: number;
}

export const apiClient = {
  // Submit a new task
  submitTask: async (params: TaskSubmitParams): Promise<TaskResponse> => {
    const response = await fetch(`${API_URL}/api/tasks/submit`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(params),
    });

    if (!response.ok) {
      throw new Error(`Failed to submit task: ${response.statusText}`);
    }

    return response.json();
  },

  // Get all tasks
  getTasks: async () => {
    const response = await fetch(`${API_URL}/api/tasks`);
    if (!response.ok) {
      throw new Error(`Failed to fetch tasks: ${response.statusText}`);
    }
    return response.json();
  },

  // Get all skills
  getSkills: async (): Promise<SkillsResponse> => {
    const response = await fetch(`${API_URL}/api/skills`);
    if (!response.ok) {
      throw new Error(`Failed to fetch skills: ${response.statusText}`);
    }
    return response.json();
  },

  // Stream task events
  streamTaskEvents: (taskID: string, onEvent: (event: any) => void) => {
    const eventSource = new EventSource(`${API_URL}/api/events?taskID=${taskID}`);

    eventSource.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data);
        onEvent(data);
      } catch (error) {
        console.error('[API] Failed to parse event:', error);
      }
    };

    eventSource.onerror = (error) => {
      console.error('[API] EventSource error:', error);
      eventSource.close();
    };

    return eventSource;
  },

  // Health check
  healthCheck: async () => {
    try {
      const response = await fetch(`${API_URL}/health`);
      return response.ok;
    } catch {
      return false;
    }
  },
};
