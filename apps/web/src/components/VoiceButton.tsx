import React, { useState } from 'react';
import { TouchableOpacity, Text, StyleSheet, Alert } from 'react-native';

interface VoiceButtonProps {
  onTranscript: (text: string) => void;
  disabled?: boolean;
}

const VoiceButton: React.FC<VoiceButtonProps> = ({ onTranscript, disabled = false }) => {
  const [listening, setListening] = useState(false);

  const startListening = async () => {
    setListening(true);
    
    // Check if Speech Recognition API is available (Web Speech API)
    if ('webkitSpeechRecognition' in window || 'SpeechRecognition' in window) {
      const SpeechRecognition = (window as any).webkitSpeechRecognition || (window as any).SpeechRecognition;
      const recognition = new SpeechRecognition();
      
      recognition.onstart = () => {
        console.log('[Voice] Listening...');
      };
      
      recognition.onresult = (event: any) => {
        let transcript = '';
        for (let i = event.resultIndex; i < event.results.length; i++) {
          transcript += event.results[i][0].transcript;
        }
        onTranscript(transcript);
      };
      
      recognition.onerror = (event: any) => {
        console.error('[Voice] Error:', event.error);
        Alert.alert('Voice Error', event.error);
      };
      
      recognition.onend = () => {
        setListening(false);
      };
      
      recognition.start();
    } else {
      Alert.alert('Not Supported', 'Speech Recognition is not supported in this browser.');
      setListening(false);
    }
  };

  const stopListening = async () => {
    setListening(false);
  };

  return (
    <TouchableOpacity
      style={[styles.button, listening && styles.buttonActive, disabled && styles.buttonDisabled]}
      onPress={listening ? stopListening : startListening}
      disabled={disabled}
    >
      <Text style={styles.buttonText}>{listening ? '🎤' : '🎙️'}</Text>
    </TouchableOpacity>
  );
};

const styles = StyleSheet.create({
  button: {
    width: 44,
    height: 44,
    borderRadius: 22,
    backgroundColor: '#1a1a1a',
    borderWidth: 1,
    borderColor: '#333',
    justifyContent: 'center',
    alignItems: 'center',
  },
  buttonActive: {
    backgroundColor: '#10a37f',
    borderColor: '#10a37f',
  },
  buttonDisabled: {
    opacity: 0.5,
  },
  buttonText: {
    fontSize: 20,
  },
});

export default VoiceButton;
