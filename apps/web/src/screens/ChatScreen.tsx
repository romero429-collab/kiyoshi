import React, { useEffect, useState } from 'react';
import { View, Text, StyleSheet, ScrollView, TouchableOpacity, TextInput, ActivityIndicator } from 'react-native';
import { useChatStore } from '../store/chatStore';
import { apiClient } from '../services/apiClient';
import ChatMessage from '../components/ChatMessage';
import VoiceButton from '../components/VoiceButton';

const ChatScreen = () => {
  const { messages, addMessage } = useChatStore();
  const [input, setInput] = useState('');
  const [loading, setLoading] = useState(false);
  const [taskStatus, setTaskStatus] = useState('idle');
  const [connected, setConnected] = useState(false);
  const [currentTaskId, setCurrentTaskId] = useState<string | null>(null);

  // Check API connection on mount
  useEffect(() => {
    const checkConnection = async () => {
      const isHealthy = await apiClient.healthCheck();
      setConnected(isHealthy);
    };
    checkConnection();
    const interval = setInterval(checkConnection, 30000); // Check every 30s
    return () => clearInterval(interval);
  }, []);

  // Stream events from task
  useEffect(() => {
    if (!currentTaskId) return;

    const eventSource = apiClient.streamTaskEvents(currentTaskId, (event) => {
      console.log('[Chat] Event:', event);
      setTaskStatus(event.status || 'executing');

      if (event.status === 'completed' || event.status === 'failed') {
        setLoading(false);
        setTaskStatus('idle');
        setCurrentTaskId(null);
      }
    });

    return () => eventSource.close();
  }, [currentTaskId]);

  const handleSubmit = async () => {
    if (!input.trim()) return;
    if (!connected) {
      addMessage({
        id: Date.now().toString(),
        role: 'assistant',
        content: '⚠️ API server not connected. Make sure the backend is running.',
        timestamp: new Date(),
      });
      return;
    }

    // Add user message
    addMessage({
      id: Date.now().toString(),
      role: 'user',
      content: input,
      timestamp: new Date(),
    });

    setInput('');
    setLoading(true);
    setTaskStatus('executing');

    try {
      const response = await apiClient.submitTask({
        title: input.substring(0, 50),
        context: input,
        difficulty: 2,
        approvalRequired: false,
      });

      console.log('[Chat] Task submitted:', response);
      setCurrentTaskId(response.taskID);

      addMessage({
        id: Date.now().toString(),
        role: 'assistant',
        content: `✅ Task ${response.taskID.substring(0, 13)}... submitted. Processing phases...`,
        timestamp: new Date(),
      });
    } catch (error) {
      console.error('[Chat] Error:', error);
      setLoading(false);
      setTaskStatus('idle');
      addMessage({
        id: Date.now().toString(),
        role: 'assistant',
        content: `❌ Error: ${(error as Error).message}`,
        timestamp: new Date(),
      });
    }
  };

  const handleVoiceInput = (text: string) => {
    setInput(text);
  };

  return (
    <View style={styles.container}>
      <View style={styles.header}>
        <View style={styles.headerLeft}>
          <Text style={styles.title}>Kiyoshi</Text>
          <View style={[styles.statusDot, connected ? styles.statusDotGreen : styles.statusDotRed]} />
          <Text style={styles.statusText}>{connected ? 'Connected' : 'Offline'}</Text>
        </View>
        <Text style={styles.status}>{taskStatus}</Text>
      </View>

      <ScrollView style={styles.messagesContainer}>
        {messages.length === 0 && (
          <View style={styles.emptyState}>
            <Text style={styles.emptyStateTitle}>Welcome to Kiyoshi</Text>
            <Text style={styles.emptyStateText}>
              Ask me anything. I'll break it into phases and execute in parallel.
            </Text>
          </View>
        )}
        {messages.map((msg) => (
          <ChatMessage key={msg.id} message={msg} />
        ))}
      </ScrollView>

      <View style={styles.inputContainer}>
        <TextInput
          style={styles.input}
          placeholder="Ask me anything..."
          value={input}
          onChangeText={setInput}
          editable={!loading && connected}
          placeholderTextColor="#666"
          multiline
        />
        <TouchableOpacity 
          style={[styles.sendButton, (loading || !connected) && styles.sendButtonDisabled]} 
          onPress={handleSubmit}
          disabled={loading || !connected}
        >
          {loading ? (
            <ActivityIndicator size="small" color="#fff" />
          ) : (
            <Text style={styles.sendButtonText}>Send</Text>
          )}
        </TouchableOpacity>
        <VoiceButton onTranscript={handleVoiceInput} disabled={loading} />
      </View>
    </View>
  );
};

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: '#0a0a0a',
  },
  header: {
    paddingTop: 16,
    paddingBottom: 12,
    paddingHorizontal: 16,
    borderBottomWidth: 1,
    borderBottomColor: '#222',
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
  },
  headerLeft: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 8,
  },
  title: {
    fontSize: 24,
    fontWeight: 'bold',
    color: '#fff',
  },
  statusDot: {
    width: 8,
    height: 8,
    borderRadius: 4,
  },
  statusDotGreen: {
    backgroundColor: '#10a37f',
  },
  statusDotRed: {
    backgroundColor: '#f85149',
  },
  statusText: {
    fontSize: 11,
    color: '#888',
  },
  status: {
    fontSize: 12,
    color: '#888',
    textTransform: 'capitalize',
  },
  messagesContainer: {
    flex: 1,
    paddingHorizontal: 16,
    paddingVertical: 12,
  },
  emptyState: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
    paddingHorizontal: 32,
    paddingVertical: 64,
  },
  emptyStateTitle: {
    fontSize: 24,
    fontWeight: 'bold',
    color: '#fff',
    marginBottom: 12,
    textAlign: 'center',
  },
  emptyStateText: {
    fontSize: 14,
    color: '#888',
    textAlign: 'center',
  },
  inputContainer: {
    paddingHorizontal: 16,
    paddingVertical: 12,
    borderTopWidth: 1,
    borderTopColor: '#222',
    flexDirection: 'row',
    alignItems: 'center',
    gap: 8,
  },
  input: {
    flex: 1,
    backgroundColor: '#1a1a1a',
    borderRadius: 8,
    paddingHorizontal: 12,
    paddingVertical: 10,
    color: '#fff',
    borderWidth: 1,
    borderColor: '#333',
    maxHeight: 100,
  },
  sendButton: {
    backgroundColor: '#10a37f',
    borderRadius: 8,
    paddingHorizontal: 16,
    paddingVertical: 10,
    minWidth: 60,
    justifyContent: 'center',
    alignItems: 'center',
  },
  sendButtonDisabled: {
    opacity: 0.5,
  },
  sendButtonText: {
    color: '#fff',
    fontWeight: '600',
  },
});

export default ChatScreen;
