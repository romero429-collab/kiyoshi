import React from 'react';
import { View, Text, StyleSheet } from 'react-native';

interface ChatMessageProps {
  message: {
    id: string;
    role: 'user' | 'assistant';
    content: string;
    timestamp: Date;
  };
}

const ChatMessage: React.FC<ChatMessageProps> = ({ message }) => {
  const isUser = message.role === 'user';

  return (
    <View style={[styles.messageWrapper, isUser ? styles.userWrapper : styles.assistantWrapper]}>
      <View style={[styles.message, isUser ? styles.userMessage : styles.assistantMessage]}>
        <Text style={styles.text}>{message.content}</Text>
        <Text style={styles.timestamp}>
          {message.timestamp.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}
        </Text>
      </View>
    </View>
  );
};

const styles = StyleSheet.create({
  messageWrapper: {
    marginVertical: 8,
  },
  userWrapper: {
    alignItems: 'flex-end',
  },
  assistantWrapper: {
    alignItems: 'flex-start',
  },
  message: {
    maxWidth: '80%',
    paddingHorizontal: 12,
    paddingVertical: 8,
    borderRadius: 8,
  },
  userMessage: {
    backgroundColor: '#10a37f',
  },
  assistantMessage: {
    backgroundColor: '#1a1a1a',
    borderWidth: 1,
    borderColor: '#333',
  },
  text: {
    color: '#fff',
    fontSize: 14,
  },
  timestamp: {
    color: '#888',
    fontSize: 11,
    marginTop: 4,
  },
});

export default ChatMessage;
