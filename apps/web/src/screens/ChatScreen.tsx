import React, { useEffect, useState } from 'react';
import { View, Text, StyleSheet, ScrollView, TouchableOpacity, TextInput } from 'react-native';
import { useChatStore } from '../store/chatStore';
import { useIPCClient } from '../services/ipcClient';
import ChatMessage from '../components/ChatMessage';
import VoiceButton from '../components/VoiceButton';

const ChatScreen = () => {
  const { messages, addMessage } = useChatStore();
  const { send } = useIPCClient();
  const [input, setInput] = useState('');
  const [loading, setLoading] = useState(false);
  const [taskStatus, setTaskStatus] = useState('idle');

  const handleSubmit = async () => {
    if (!input.trim()) return;

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
      // Send task to CLI
      const response = await send('task.submit', {
        title: input.substring(0, 50),
        context: input,
        difficulty: 2,
        approvalRequired: false,
      });

      console.log('[Chat] Task submitted:', response);

      // Add assistant response
      addMessage({
        id: Date.now().toString(),
        role: 'assistant',
        content: `Task submitted: ${response.taskID}. Processing...`,
        timestamp: new Date(),
      });
    } catch (error) {
      console.error('[Chat] Error:', error);
      addMessage({
        id: Date.now().toString(),
        role: 'assistant',
        content: `Error: ${error.message}`,
        timestamp: new Date(),
      });
    } finally {
      setLoading(false);
      setTaskStatus('idle');
    }
  };

  const handleVoiceInput = (text) => {
    setInput(text);
  };

  return (
    <View style={styles.container}>
      <View style={styles.header}>
        <Text style={styles.title}>Kiyoshi</Text>
        <Text style={styles.status}>{taskStatus}</Text>
      </View>

      <ScrollView style={styles.messagesContainer}>
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
          editable={!loading}
          placeholderTextColor="#666"
        />
        <TouchableOpacity 
          style={[styles.sendButton, loading && styles.sendButtonDisabled]} 
          onPress={handleSubmit}
          disabled={loading}
        >
          <Text style={styles.sendButtonText}>Send</Text>
        </TouchableOpacity>
        <VoiceButton onTranscript={handleVoiceInput} />
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
  title: {
    fontSize: 24,
    fontWeight: 'bold',
    color: '#fff',
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
  },
  sendButton: {
    backgroundColor: '#10a37f',
    borderRadius: 8,
    paddingHorizontal: 16,
    paddingVertical: 10,
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
