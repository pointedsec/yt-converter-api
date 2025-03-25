import yt_dlp
import sys
import json
import os
import subprocess

def convert_m4a_to_mp3(m4a_path: str, mp3_path: str) -> str:
    try:
        # Use ffmpeg to convert M4A to MP3
        subprocess.run([
            'ffmpeg', '-i', m4a_path, '-codec:a', 'libmp3lame', '-q:a', '2', mp3_path
        ], check=True)
        return mp3_path
    except subprocess.CalledProcessError as e:
        print(f"Error converting M4A to MP3: {e}")
        return ""
    
def convert_to_audio(youtube_url: str, output_path: str, youtube_id: str) -> str:
    ydl_opts = {
    'format': 'm4a/bestaudio/best',
    'postprocessors': [{
        'key': 'FFmpegExtractAudio',
        'preferredcodec': 'm4a',
    }],
    'outtmpl': output_path + '/%(id)s.%(ext)s',
    }
    with yt_dlp.YoutubeDL(ydl_opts) as ydl:
        error_code = ydl.download(youtube_url)
        return output_path + f'/{youtube_id}.m4a'

def get_video_available_resolutions(youtube_url: str) -> list[str]:
    ydl_opts = {
        'quiet': True,
    }

    with yt_dlp.YoutubeDL(ydl_opts) as ydl:
        info = ydl.extract_info(youtube_url, download=False)
        streams = info.get('formats', [])
        resolutions = set()

        for stream in streams:
            if stream.get('vcodec') not in [None, 'none']:
                height = stream.get('height')
                if height:
                    resolutions.add(f"{height}p")

        return sorted(resolutions, key=lambda x: int(x.replace('p', '')))

def convert_to_video(youtube_url: str, resolution: str, output_path: str) -> str:
    available_resolutions = get_video_available_resolutions(youtube_url)
    
    if resolution not in available_resolutions:
        print(f"❌ Resolution {resolution} not available. Available resolutions: {available_resolutions}")
        return ""
    
    print(f"⏳ Downloading video {youtube_url} with resolution {resolution}...")
    ydl_opts = {
        'format': f'bestvideo[height={resolution[:-1]}]+bestaudio/best',
        'outtmpl': output_path + '/%(id)s-%(resolution)s.%(ext)s',
        'merge_output_format': 'mp4',
    }
    
    with yt_dlp.YoutubeDL(ydl_opts) as ydl:
        try:
            info_dict = ydl.extract_info(youtube_url, download=True)
            output_filename = ydl.prepare_filename(info_dict)
            return output_filename
        except Exception as e:
            print(f"Error downloading video: {e}")
            return ""

def video_to_youtube_url(video_id: str) -> str:
    return f"https://www.youtube.com/watch?v={video_id}"

def delete_repeated_resolutions(resolutions: list[str]) -> list[str]:
    return list(set(resolutions))

def check_if_path_is_valid_and_absolute(path: str) -> bool:
    return os.path.exists(path) and os.path.isabs(path)

if __name__ == "__main__":

    if len(sys.argv) < 4:
        print("Please provide a video ID, convert to format (audio/video) and output path")
        exit(1)

    # Get Video ID from Parameter
    video_id = sys.argv[1]

    # Get If you want to convert to audio or video
    convert_to = sys.argv[2] 

    output_path = sys.argv[3]

    if not check_if_path_is_valid_and_absolute(output_path):
        print("Please provide a valid and absolute output path")
        exit(1)

    if video_id is None:
        print("Please provide a video ID")
        exit(1)

    if convert_to is None or convert_to not in ["audio", "video"]:
        print("Please provide a convert to format (audio/video)")
        exit(1)

    if output_path is None:
        print("Please provide a output path")
        exit(1)

    # Get Video URL
    video_url = video_to_youtube_url(video_id)

    if convert_to == "audio":
        path = convert_to_audio(video_url, output_path, video_id)
        mp3path = convert_m4a_to_mp3(path, output_path + f'/{video_id}.mp3')
        os.remove(path)
        print(mp3path)
        exit(0)
    elif convert_to == "video":
        try:
            resolution = sys.argv[4]
        except IndexError:
            resolution = None
        if resolution is None:
            resolutions = get_video_available_resolutions(video_url)
            print(resolutions)
            exit(0)
        path = convert_to_video(video_url, resolution, output_path)
        print(path)
        exit(0)