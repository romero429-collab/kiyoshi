export interface Task {
  id: string;
  title: string;
  context: string;
  difficulty: number;
  status: TaskStatus;
  phases: Phase[];
  startTime: Date;
  endTime?: Date;
}

export type TaskStatus = 'planning' | 'executing' | 'completed' | 'failed' | 'cancelled';

export interface Phase {
  id: string;
  title: string;
  type: PhaseType;
  dependencies: string[];
  agentID: string;
  status: PhaseStatus;
  output: any;
}

export type PhaseType = 'analysis' | 'generation' | 'validation' | 'execution' | 'logging';
export type PhaseStatus = 'pending' | 'ready' | 'running' | 'completed' | 'failed' | 'skipped';

export interface Skill {
  id: string;
  title: string;
  category: SkillCategory;
  difficulty: number;
  tags: string[];
  description: string;
  example: string;
  sourceTask: string;
  createdAt: Date;
  usedCount: number;
  successRate: number;
}

export type SkillCategory = 'code_pattern' | 'decision_tree' | 'error_fix' | 'optimization' | 'best_practice';

export interface Agent {
  id: string;
  taskID: string;
  phaseID: string;
  type: AgentType;
  status: AgentStatus;
  input: any;
  output: any;
}

export type AgentType = 'analyzer' | 'generator' | 'validator' | 'logger';
export type AgentStatus = 'initializing' | 'ready' | 'running' | 'completed' | 'failed';

export interface Event {
  type: EventType;
  taskID: string;
  phaseID?: string;
  agentID?: string;
  timestamp: Date;
  data: any;
}

export type EventType = 
  | 'task.started'
  | 'task.completed'
  | 'task.failed'
  | 'phase.started'
  | 'phase.progress'
  | 'phase.completed'
  | 'phase.failed';
